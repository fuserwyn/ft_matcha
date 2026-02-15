package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
)

const IndexName = "matcha_users"

type UserDoc struct {
	UserID           string   `json:"user_id"`
	Username         string   `json:"username"`
	FirstName        string   `json:"first_name"`
	LastName         string   `json:"last_name"`
	Gender           string   `json:"gender,omitempty"`
	SexualPreference string   `json:"sexual_preference,omitempty"`
	BirthDate        string   `json:"birth_date,omitempty"`
	Bio              string   `json:"bio,omitempty"`
	FameRating       int      `json:"fame_rating"`
	Location         *GeoPoint `json:"location,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type SearchFilters struct {
	ExcludeID   uuid.UUID
	Gender      string
	Interest    string
	MinAge      int
	MaxAge      int
	Limit       int
	Offset      int
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

	must := []map[string]interface{}{
		{"bool": map[string]interface{}{
			"must_not": []map[string]interface{}{
				{"term": map[string]interface{}{"user_id": f.ExcludeID.String()}},
			},
		}},
	}
	if f.Gender != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"gender": f.Gender},
		})
	}
	if f.Interest != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"sexual_preference": f.Interest},
		})
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

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{"must": must},
		},
		"sort": []map[string]interface{}{
			{"fame_rating": map[string]interface{}{"order": "desc"}},
			{"created_at": map[string]interface{}{"order": "desc"}},
		},
		"from": f.Offset,
		"size": f.Limit,
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
