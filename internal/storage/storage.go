package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"dynamic-form-engine/internal/models"

	"github.com/boltdb/bolt"
	"github.com/google/uuid"
)

var (
	bucketSchemas     = []byte("schemas")
	bucketSubmissions = []byte("submissions")
	bucketRateLimit   = []byte("ratelimit")
)

type Store struct {
	db *bolt.DB
}

func NewStore(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketSchemas)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketSubmissions)
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists(bucketRateLimit)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateSchema(schema *models.FormSchema) error {
	if schema.ID == "" {
		schema.ID = uuid.New().String()
	}
	schema.Version = 1
	now := time.Now()
	schema.CreatedAt = now
	schema.UpdatedAt = now

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSchemas)

		key := []byte(schema.ID + ":v1")
		data, err := json.Marshal(schema)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

func (s *Store) UpdateSchema(schema *models.FormSchema) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSchemas)

		latest, err := s.getLatestSchemaLocked(tx, schema.ID)
		if err != nil {
			return err
		}
		if latest == nil {
			return fmt.Errorf("schema not found: %s", schema.ID)
		}

		newVersion := latest.Version + 1
		schema.Version = newVersion
		schema.CreatedAt = latest.CreatedAt
		schema.UpdatedAt = time.Now()

		key := []byte(fmt.Sprintf("%s:v%d", schema.ID, newVersion))
		data, err := json.Marshal(schema)
		if err != nil {
			return err
		}
		return b.Put(key, data)
	})
}

func (s *Store) GetSchema(id string, version int) (*models.FormSchema, error) {
	var schema models.FormSchema
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSchemas)
		var key []byte
		if version > 0 {
			key = []byte(fmt.Sprintf("%s:v%d", id, version))
		} else {
			latest, err := s.getLatestSchemaLocked(tx, id)
			if err != nil {
				return err
			}
			if latest == nil {
				return nil
			}
			key = []byte(fmt.Sprintf("%s:v%d", id, latest.Version))
		}
		data := b.Get(key)
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &schema)
	})
	if err != nil {
		return nil, err
	}
	if schema.ID == "" {
		return nil, nil
	}
	return &schema, nil
}

func (s *Store) getLatestSchemaLocked(tx *bolt.Tx, id string) (*models.FormSchema, error) {
	b := tx.Bucket(bucketSchemas)
	prefix := []byte(id + ":")

	var versions []int
	c := b.Cursor()
	for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
		verStr := string(k[len(prefix)+1:])
		var v int
		if _, err := fmt.Sscanf(verStr, "%d", &v); err == nil {
			versions = append(versions, v)
		}
	}

	if len(versions) == 0 {
		return nil, nil
	}

	sort.Sort(sort.Reverse(sort.IntSlice(versions)))
	latestKey := []byte(fmt.Sprintf("%s:v%d", id, versions[0]))
	data := b.Get(latestKey)
	if data == nil {
		return nil, nil
	}

	var schema models.FormSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

func (s *Store) GetLatestSchema(id string) (*models.FormSchema, error) {
	var schema *models.FormSchema
	err := s.db.View(func(tx *bolt.Tx) error {
		s, err := s.getLatestSchemaLocked(tx, id)
		schema = s
		return err
	})
	return schema, err
}

func (s *Store) ListSchemas() ([]models.FormSchema, error) {
	var schemas []models.FormSchema
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSchemas)
		c := b.Cursor()

		seen := make(map[string]bool)
		for k, v := c.Last(); k != nil; k, _ = c.Prev() {
			keyStr := string(k)
			var id, verStr string
			fmt.Sscanf(keyStr, "%[^:]:v%s", &id, &verStr)
			if seen[id] {
				continue
			}
			seen[id] = true

			var schema models.FormSchema
			if err := json.Unmarshal(v, &schema); err != nil {
				continue
			}
			schemas = append(schemas, schema)
		}
		return nil
	})
	return schemas, err
}

func (s *Store) DeleteSchema(id string) (bool, error) {
	deleted := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSchemas)
		prefix := []byte(id + ":")
		c := b.Cursor()
		for k, _ := c.Seek(prefix); k != nil && len(k) >= len(prefix) && string(k[:len(prefix)]) == string(prefix); k, _ = c.Next() {
			if err := b.Delete(append([]byte{}, k...)); err != nil {
				return err
			}
			deleted = true
		}
		return nil
	})
	return deleted, err
}

func (s *Store) CreateSubmission(sub *models.Submission) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	sub.Revision = 1

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSubmissions)
		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}
		return b.Put([]byte(sub.ID), data)
	})
}

func (s *Store) UpdateSubmission(sub *models.Submission) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSubmissions)
		existing := b.Get([]byte(sub.ID))
		if existing == nil {
			return fmt.Errorf("submission not found: %s", sub.ID)
		}

		var stored models.Submission
		if err := json.Unmarshal(existing, &stored); err != nil {
			return err
		}
		if stored.Revision != sub.Revision {
			return models.ErrConflict
		}

		sub.Revision = stored.Revision + 1
		sub.UpdatedAt = time.Now()

		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}
		return b.Put([]byte(sub.ID), data)
	})
}

func (s *Store) GetSubmission(id string) (*models.Submission, error) {
	var sub models.Submission
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSubmissions)
		data := b.Get([]byte(id))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &sub)
	})
	if err != nil {
		return nil, err
	}
	if sub.ID == "" {
		return nil, nil
	}
	return &sub, nil
}

func (s *Store) ListSubmissions(formID string) ([]models.Submission, error) {
	var subs []models.Submission
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSubmissions)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var sub models.Submission
			if err := json.Unmarshal(v, &sub); err != nil {
				continue
			}
			if formID == "" || sub.FormID == formID {
				subs = append(subs, sub)
			}
		}
		return nil
	})

	sort.Slice(subs, func(i, j int) bool {
		return subs[i].CreatedAt.After(subs[j].CreatedAt)
	})

	return subs, err
}

func (s *Store) DeleteSubmission(id string) (bool, error) {
	deleted := false
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSubmissions)
		key := []byte(id)
		existing := b.Get(key)
		if existing == nil {
			return nil
		}
		deleted = true
		return b.Delete(key)
	})
	return deleted, err
}

func (s *Store) RecordSubmissionAttempt(formID, userID string, window time.Duration, maxCount int) (bool, error) {
	now := time.Now()
	minuteKey := now.Truncate(window).Format(time.RFC3339)
	key := fmt.Sprintf("%s:%s:%s", formID, userID, minuteKey)

	var allowed bool
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketRateLimit)
		data := b.Get([]byte(key))
		var count int
		if data != nil {
			json.Unmarshal(data, &count)
		}

		if count >= maxCount {
			allowed = false
			return nil
		}

		count++
		data, _ = json.Marshal(count)
		if err := b.Put([]byte(key), data); err != nil {
			return err
		}
		allowed = true
		return nil
	})
	return allowed, err
}

func (s *Store) CleanupOldRateLimit(cutoff time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketRateLimit)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			keyStr := string(k)
			var formID, userID, timeStr string
			_, err := fmt.Sscanf(keyStr, "%[^:]:%[^:]:%s", &formID, &userID, &timeStr)
			if err != nil {
				continue
			}
			t, err := time.Parse(time.RFC3339, timeStr)
			if err != nil {
				continue
			}
			if t.Before(cutoff) {
				b.Delete(append([]byte{}, k...))
			}
		}
		return nil
	})
}
