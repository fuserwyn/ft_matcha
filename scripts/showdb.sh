#!/usr/bin/env bash
set -euo pipefail

CONTAINER=$(docker ps --format '{{.Names}}' | grep -E 'postgres' | head -1)

if [ -z "$CONTAINER" ]; then
  echo "ERROR: No running postgres container found." >&2
  exit 1
fi

echo "=== Connected to: $CONTAINER ==="

psql_x() {
  docker exec -i "$CONTAINER" psql -U matcha -d matcha -P pager=off "$@"
}

sep() {
  echo ""
  printf '%.0s═' {1..60}; echo ""
  echo "  $1"
  printf '%.0s═' {1..60}; echo ""
}

sep "USERS"
psql_x -c "
SELECT id, username, email, first_name, last_name,
       email_verified_at IS NOT NULL AS verified,
       password_hash,
       created_at
FROM users
ORDER BY created_at DESC
LIMIT 50;"

sep "PROFILES (gender / bio / location / fame)"
psql_x -c "
SELECT u.username,
       p.gender,
       p.sexual_preference AS pref,
       p.birth_date,
       p.city,
       ROUND(p.latitude::numeric,2)  AS lat,
       ROUND(p.longitude::numeric,2) AS lng,
       p.fame_rating AS fame,
       LEFT(p.bio, 40) AS bio
FROM profiles p
JOIN users u ON u.id = p.user_id
ORDER BY u.username
LIMIT 50;"

sep "PHOTOS"
psql_x -c "
SELECT up.id, u.username, up.is_primary, up.position, up.url, up.created_at
FROM user_photos up
JOIN users u ON u.id = up.user_id
ORDER BY u.username, up.position
LIMIT 100;"

sep "TAGS  (usage count)"
psql_x -c "
SELECT t.name, COUNT(ut.user_id) AS users
FROM tags t
LEFT JOIN user_tags ut ON ut.tag_id = t.id
GROUP BY t.name
ORDER BY users DESC, t.name;"

sep "USER TAGS"
psql_x -c "
SELECT u.username, t.name AS tag
FROM user_tags ut
JOIN users u ON u.id = ut.user_id
JOIN tags t  ON t.id = ut.tag_id
ORDER BY u.username, t.name
LIMIT 100;"

sep "LIKES"
psql_x -c "
SELECT u1.username AS from_user, u2.username AS to_user, l.created_at
FROM likes l
JOIN users u1 ON u1.id = l.user_id
JOIN users u2 ON u2.id = l.liked_user_id
ORDER BY l.created_at DESC
LIMIT 50;"

sep "MATCHES  (mutual likes)"
psql_x -c "
SELECT u1.username AS user_a, u2.username AS user_b, a.created_at AS matched_at
FROM likes a
JOIN likes b ON a.user_id = b.liked_user_id AND a.liked_user_id = b.user_id
JOIN users u1 ON u1.id = a.user_id
JOIN users u2 ON u2.id = a.liked_user_id
WHERE a.user_id < a.liked_user_id
ORDER BY matched_at DESC
LIMIT 50;"

sep "MESSAGES"
psql_x -c "
SELECT u1.username AS sender, u2.username AS receiver,
       LEFT(m.content, 60) AS preview,
       m.is_read, m.created_at
FROM messages m
JOIN users u1 ON u1.id = m.sender_id
JOIN users u2 ON u2.id = m.receiver_id
ORDER BY m.created_at DESC
LIMIT 50;"

sep "NOTIFICATIONS"
psql_x -c "
SELECT u.username AS for_user,
       a.username AS actor,
       n.type, n.is_read,
       LEFT(n.content, 50) AS content,
       n.created_at
FROM notifications n
JOIN users u ON u.id = n.user_id
LEFT JOIN users a ON a.id = n.actor_id
ORDER BY n.created_at DESC
LIMIT 50;"

sep "PROFILE VIEWS"
psql_x -c "
SELECT u1.username AS viewer, u2.username AS viewed, pv.created_at
FROM profile_views pv
JOIN users u1 ON u1.id = pv.viewer_user_id
JOIN users u2 ON u2.id = pv.viewed_user_id
ORDER BY pv.created_at DESC
LIMIT 50;"

sep "BLOCKS"
psql_x -c "
SELECT u1.username AS blocker, u2.username AS blocked, ub.created_at
FROM user_blocks ub
JOIN users u1 ON u1.id = ub.blocker_user_id
JOIN users u2 ON u2.id = ub.blocked_user_id
ORDER BY ub.created_at DESC
LIMIT 50;"

sep "REPORTS"
psql_x -c "
SELECT u1.username AS reporter, u2.username AS target,
       ur.reason, ur.status, LEFT(ur.comment,40) AS comment, ur.created_at
FROM user_reports ur
JOIN users u1 ON u1.id = ur.reporter_user_id
JOIN users u2 ON u2.id = ur.target_user_id
ORDER BY ur.created_at DESC
LIMIT 50;"

sep "PRESENCE  (last seen)"
psql_x -c "
SELECT u.username, up.last_seen,
       CASE WHEN up.last_seen > now() - interval '5 minutes'
            THEN 'online' ELSE 'offline' END AS status
FROM user_presence up
JOIN users u ON u.id = up.user_id
ORDER BY up.last_seen DESC
LIMIT 50;"

echo ""
echo "Done."
