package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
)

const IndexName = "matcha_users"

type UserDoc struct {
	UserID           string    `json:"user_id"`
	Username         string    `json:"username"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Gender           string    `json:"gender,omitempty"`
	SexualPreference string    `json:"sexual_preference,omitempty"`
	BirthDate        string    `json:"birth_date,omitempty"`
	Bio              string    `json:"bio,omitempty"`
	City             string    `json:"city,omitempty"`
	Tags             []string  `json:"tags,omitempty"`
	FameRating       int       `json:"fame_rating"`
	Location         *GeoPoint `json:"location,omitempty"`
	CreatedAt        string    `json:"created_at"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type SearchFilters struct {
	ExcludeID     uuid.UUID
	ExcludeIDs    []uuid.UUID
	Genders       []string // multiple: male, female, etc.
	Interests     []string // multiple: male, female, both, etc.
	Tags          []string
	StrictTags    bool
	City          string
	PreferredCity string
	MinAge        int
	MaxAge        int
	MinFame       int
	MaxFame       int
	UserLat       *float64
	UserLon       *float64
	MaxDistanceKm int
	SortBy        string
	SortOrder     string
	Limit         int
	Offset        int
}

type Client struct {
	es *elasticsearch.Client
}

func NewClient(cfg elasticsearch.Config) (*Client, error) {
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{es: es}, nil
}

func (c *Client) EnsureIndex(ctx context.Context) error {
	existsReq := esapi.IndicesExistsRequest{Index: []string{IndexName}}
	existsRes, err := existsReq.Do(ctx, c.es)
	if err != nil {
		return err
	}
	existsRes.Body.Close()
	if existsRes.StatusCode == 200 {
		return nil
	}
	mapping := `{
		"mappings": {
			"properties": {
				"user_id": { "type": "keyword" },
				"username": { "type": "keyword" },
				"first_name": { "type": "text" },
				"last_name": { "type": "text" },
				"gender": { "type": "keyword" },
				"sexual_preference": { "type": "keyword" },
				"birth_date": { "type": "date", "format": "yyyy-MM-dd" },
				"bio": { "type": "text" },
				"city": { "type": "keyword" },
				"tags": { "type": "keyword" },
				"fame_rating": { "type": "integer" },
				"location": { "type": "geo_point" },
				"created_at": { "type": "date" }
			}
		}
	}`
	req := esapi.IndicesCreateRequest{
		Index: IndexName,
		Body:  bytes.NewReader([]byte(mapping)),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("create index: %s", res.String())
	}
	return nil
}

func (c *Client) Index(ctx context.Context, doc *UserDoc) error {
	doc.CreatedAt = time.Now().Format(time.RFC3339)
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	req := esapi.IndexRequest{
		Index:      IndexName,
		DocumentID: doc.UserID,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("index: %s", res.String())
	}
	return nil
}

func (c *Client) SearchCities(ctx context.Context, prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, nil
	}
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"prefix": map[string]interface{}{
							"city.keyword": map[string]interface{}{
								"value":            prefix,
								"case_insensitive": true,
							},
						},
					},
					{"exists": map[string]interface{}{"field": "city.keyword"}},
				},
			},
		},
		"aggs": map[string]interface{}{
			"cities": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "city.keyword",
					"size":  limit,
					"order": map[string]interface{}{"_key": "asc"},
				},
			},
		},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{IndexName},
		Body:  bytes.NewReader(body),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("search cities: %s", res.String())
	}
	var result struct {
		Aggregations struct {
			Cities struct {
				Buckets []struct {
					Key string `json:"key"`
				} `json:"buckets"`
			} `json:"cities"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	cities := make([]string, 0, len(result.Aggregations.Cities.Buckets))
	for _, b := range result.Aggregations.Cities.Buckets {
		if b.Key != "" {
			cities = append(cities, b.Key)
		}
	}
	return cities, nil
}

func (c *Client) SearchTags(ctx context.Context, prefix string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix == "" {
		return nil, nil
	}
	// Regex for tags starting with prefix (case-insensitive)
	includePattern := "(?i)" + regexp.QuoteMeta(prefix) + ".*"
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"exists": map[string]interface{}{"field": "tags"},
		},
		"aggs": map[string]interface{}{
			"tags": map[string]interface{}{
				"terms": map[string]interface{}{
					"field":    "tags",
					"size":     limit,
					"order":    map[string]interface{}{"_key": "asc"},
					"include":  includePattern,
				},
			},
		},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{IndexName},
		Body:  bytes.NewReader(body),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("search tags: %s", res.String())
	}
	var result struct {
		Aggregations struct {
			Tags struct {
				Buckets []struct {
					Key string `json:"key"`
				} `json:"buckets"`
			} `json:"tags"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(result.Aggregations.Tags.Buckets))
	for _, b := range result.Aggregations.Tags.Buckets {
		if b.Key != "" {
			tags = append(tags, b.Key)
		}
	}
	return tags, nil
}

func (c *Client) FilterAggregations(ctx context.Context, excludeID uuid.UUID, excludeIDs []uuid.UUID) (gender map[string]int64, interest map[string]int64, err error) {
	mustNot := []map[string]interface{}{
		{"term": map[string]interface{}{"user_id": excludeID.String()}},
	}
	for _, id := range excludeIDs {
		if id != uuid.Nil {
			mustNot = append(mustNot, map[string]interface{}{
				"term": map[string]interface{}{"user_id": id.String()},
			})
		}
	}
	query := map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{"must_not": mustNot},
		},
		"aggs": map[string]interface{}{
			"gender": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "gender",
					"size":  20,
					"order": map[string]interface{}{"_key": "asc"},
				},
			},
			"sexual_preference": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "sexual_preference",
					"size":  20,
					"order": map[string]interface{}{"_key": "asc"},
				},
			},
		},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, nil, err
	}
	req := esapi.SearchRequest{
		Index: []string{IndexName},
		Body:  bytes.NewReader(body),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, nil, fmt.Errorf("filter aggregations: %s", res.String())
	}
	var result struct {
		Aggregations struct {
			Gender struct {
				Buckets []struct {
					Key   string `json:"key"`
					Count int64  `json:"doc_count"`
				} `json:"buckets"`
			} `json:"gender"`
			SexualPreference struct {
				Buckets []struct {
					Key   string `json:"key"`
					Count int64  `json:"doc_count"`
				} `json:"buckets"`
			} `json:"sexual_preference"`
		} `json:"aggregations"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, nil, err
	}
	gender = make(map[string]int64)
	for _, b := range result.Aggregations.Gender.Buckets {
		if b.Key != "" {
			gender[b.Key] = b.Count
		}
	}
	interest = make(map[string]int64)
	for _, b := range result.Aggregations.SexualPreference.Buckets {
		if b.Key != "" {
			interest[b.Key] = b.Count
		}
	}
	return gender, interest, nil
}

func (c *Client) Delete(ctx context.Context, userID string) error {
	req := esapi.DeleteRequest{
		Index:      IndexName,
		DocumentID: userID,
		Refresh:    "true",
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete: %s", res.String())
	}
	return nil
}

func (c *Client) Search(ctx context.Context, f SearchFilters) ([]UserDoc, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}

	must := []map[string]interface{}{}
	mustNot := []map[string]interface{}{
		{"term": map[string]interface{}{"user_id": f.ExcludeID.String()}},
	}
	for _, id := range f.ExcludeIDs {
		if id != uuid.Nil {
			mustNot = append(mustNot, map[string]interface{}{
				"term": map[string]interface{}{"user_id": id.String()},
			})
		}
	}
	if len(f.Genders) > 0 {
		if len(f.Genders) == 1 {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{"gender": f.Genders[0]},
			})
		} else {
			must = append(must, map[string]interface{}{
				"terms": map[string]interface{}{"gender": f.Genders},
			})
		}
	}
	if len(f.Interests) > 0 {
		if len(f.Interests) == 1 {
			must = append(must, map[string]interface{}{
				"term": map[string]interface{}{"sexual_preference": f.Interests[0]},
			})
		} else {
			must = append(must, map[string]interface{}{
				"terms": map[string]interface{}{"sexual_preference": f.Interests},
			})
		}
	}
	if f.City != "" {
		// Partial match: "Par" -> Paris, "Amster" -> Amsterdam (case-insensitive)
		wildcardVal := strings.TrimSpace(f.City) + "*"
		must = append(must, map[string]interface{}{
			"wildcard": map[string]interface{}{
				"city": map[string]interface{}{
					"value":            wildcardVal,
					"case_insensitive": true,
				},
			},
		})
	}
	if len(f.Tags) > 0 && f.StrictTags {
		// Partial match: "mus" finds music, musician, etc. (case-insensitive)
		for _, tag := range f.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if tag == "" {
				continue
			}
			wildcardVal := tag + "*"
			must = append(must, map[string]interface{}{
				"wildcard": map[string]interface{}{
					"tags": map[string]interface{}{
						"value":            wildcardVal,
						"case_insensitive": true,
					},
				},
			})
		}
	}
	if f.MinAge > 0 {
		maxBirth := time.Now().AddDate(-f.MinAge, 0, 0).Format("2006-01-02")
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"birth_date": map[string]interface{}{"lte": maxBirth},
			},
		})
	}
	if f.MaxAge > 0 {
		minBirth := time.Now().AddDate(-f.MaxAge-1, 0, 0).Format("2006-01-02")
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{
				"birth_date": map[string]interface{}{"gte": minBirth},
			},
		})
	}
	if f.MinFame > 0 || f.MaxFame > 0 {
		rng := map[string]interface{}{}
		if f.MinFame > 0 {
			rng["gte"] = f.MinFame
		}
		if f.MaxFame > 0 {
			rng["lte"] = f.MaxFame
		}
		must = append(must, map[string]interface{}{
			"range": map[string]interface{}{"fame_rating": rng},
		})
	}
	if f.MaxDistanceKm > 0 && f.UserLat != nil && f.UserLon != nil {
		must = append(must, map[string]interface{}{
			"geo_distance": map[string]interface{}{
				"distance": fmt.Sprintf("%dkm", f.MaxDistanceKm),
				"location": map[string]interface{}{
					"lat": *f.UserLat,
					"lon": *f.UserLon,
				},
			},
		})
	}

	sortOrder := "desc"
	if f.SortOrder == "asc" {
		sortOrder = "asc"
	}
	sort := []map[string]interface{}{{"fame_rating": map[string]interface{}{"order": "desc"}}, {"created_at": map[string]interface{}{"order": "desc"}}}
	queryBody := map[string]interface{}{
		"bool": map[string]interface{}{"must": must, "must_not": mustNot},
	}
	if f.SortBy == "" {
		functions := []map[string]interface{}{
			{
				"field_value_factor": map[string]interface{}{
					"field":    "fame_rating",
					"factor":   0.2,
					"modifier": "sqrt",
					"missing":  0,
				},
				"weight": 2.0,
			},
		}
		if f.UserLat != nil && f.UserLon != nil {
			functions = append(functions, map[string]interface{}{
				"gauss": map[string]interface{}{
					"location": map[string]interface{}{
						"origin": map[string]interface{}{
							"lat": *f.UserLat,
							"lon": *f.UserLon,
						},
						"scale": "40km",
					},
				},
				"weight": 3.0,
			})
		}
		if f.PreferredCity != "" {
			functions = append(functions, map[string]interface{}{
				"filter": map[string]interface{}{
					"term": map[string]interface{}{"city": f.PreferredCity},
				},
				"weight": 1.5,
			})
		}
		if len(f.Tags) > 0 && !f.StrictTags {
			for _, tag := range f.Tags {
				functions = append(functions, map[string]interface{}{
					"filter": map[string]interface{}{
						"term": map[string]interface{}{"tags": tag},
					},
					"weight": 1.0,
				})
			}
		}
		queryBody = map[string]interface{}{
			"function_score": map[string]interface{}{
				"query":      queryBody,
				"functions":  functions,
				"score_mode": "sum",
				"boost_mode": "sum",
			},
		}
		sort = []map[string]interface{}{
			{"_score": map[string]interface{}{"order": "desc"}},
			{"fame_rating": map[string]interface{}{"order": "desc"}},
		}
	}
	switch f.SortBy {
	case "age":
		// older age = earlier birth_date; asc = younger first, desc = older first
		if sortOrder == "asc" {
			sort = []map[string]interface{}{
				{"birth_date": map[string]interface{}{"order": "desc"}},
				{"fame_rating": map[string]interface{}{"order": "desc"}},
			}
		} else {
			sort = []map[string]interface{}{
				{"birth_date": map[string]interface{}{"order": "asc"}},
				{"fame_rating": map[string]interface{}{"order": "desc"}},
			}
		}
	case "fame":
		sort = []map[string]interface{}{
			{"fame_rating": map[string]interface{}{"order": sortOrder}},
			{"created_at": map[string]interface{}{"order": "desc"}},
		}
	case "location":
		if f.UserLat != nil && f.UserLon != nil {
			sort = []map[string]interface{}{
				{
					"_geo_distance": map[string]interface{}{
						"location": map[string]interface{}{"lat": *f.UserLat, "lon": *f.UserLon},
						"order":    sortOrder,
						"unit":     "km",
					},
				},
				{"fame_rating": map[string]interface{}{"order": "desc"}},
			}
		} else {
			// Fallback when user has no location: sort by fame
			sort = []map[string]interface{}{
				{"fame_rating": map[string]interface{}{"order": sortOrder}},
				{"created_at": map[string]interface{}{"order": "desc"}},
			}
		}
	case "tags":
		// Sort by number of matching tags: profiles with more common tags first
		if len(f.Tags) > 0 {
			tagFunctions := make([]map[string]interface{}, 0, len(f.Tags))
			for _, tag := range f.Tags {
				tagFunctions = append(tagFunctions, map[string]interface{}{
					"filter": map[string]interface{}{
						"term": map[string]interface{}{"tags": tag},
					},
					"weight": 10.0,
				})
			}
			queryBody = map[string]interface{}{
				"function_score": map[string]interface{}{
					"query":      queryBody,
					"functions":  tagFunctions,
					"score_mode": "sum",
					"boost_mode": "sum",
				},
			}
			order := "desc"
			if sortOrder == "asc" {
				order = "asc"
			}
			sort = []map[string]interface{}{
				{"_score": map[string]interface{}{"order": order}},
				{"fame_rating": map[string]interface{}{"order": "desc"}},
			}
		} else {
			// Fallback when user has no tags: sort by fame
			sort = []map[string]interface{}{
				{"fame_rating": map[string]interface{}{"order": sortOrder}},
				{"created_at": map[string]interface{}{"order": "desc"}},
			}
		}
	}

	query := map[string]interface{}{
		"query": queryBody,
		"sort":  sort,
		"from":  f.Offset,
		"size":  f.Limit,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	req := esapi.SearchRequest{
		Index: []string{IndexName},
		Body:  bytes.NewReader(body),
	}
	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("search: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source UserDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	docs := make([]UserDoc, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		docs[i] = h.Source
	}
	return docs, nil
}
