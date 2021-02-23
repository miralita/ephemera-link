package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	"go.etcd.io/bbolt"
	"log"
	"os"
	"sync"
	"time"
)

type Secret struct {
	Id string
	Secret    []byte
	Created   time.Time
}

func (s *Secret) TimeKey() []byte {
	return []byte(s.Created.Format(time.RFC3339) + s.Id)
}

type Storage struct {
	cfg      *Config
	data     map[string]*Secret
	cron     *gocron.Scheduler
	sync.RWMutex
}

var TimeKeysName = []byte("timekeys")
var ValueKeysName = []byte("valuekeys")

func NewStorage(cfg *Config) *Storage {
	st := &Storage{cfg: cfg, data: make(map[string]*Secret), cron: gocron.NewScheduler(time.UTC)}
	st.cron.Every(30 * time.Minute).Do(st.clearExpired)
	if cfg.PersistentStorage {
		_ = os.Mkdir(cfg.StoragePath, 0700)
		err, db := st.openDb()
		if err != nil {
			log.Fatalf("Can't open database: %v", err)
		}
		defer db.Close()
		db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(TimeKeysName)
			if err != nil {
				log.Fatalf("Can't init time keys bucket: %v", err)
			}
			_, err = tx.CreateBucketIfNotExists(ValueKeysName)
			if err != nil {
				log.Fatalf("Can't init value keys bucket: %v", err)
			}
			return err
		})

	}
	return st
}

func (s *Storage) SaveSecret(secret string) (err error, id, key string) {
	key = RandString(s.cfg.KeyLength)
	err, data := Encrypt(s.cfg.KeyPart + key, secret)
	if err != nil {
		return
	}
	sec := &Secret{
		Secret:    data,
		Created:   time.Now(),
	}
	if s.cfg.PersistentStorage {
		err =  s.savePersistent(sec)
	} else {
		err =  s.saveLocal(sec)
	}
	id = sec.Id
	return
}

func (s *Storage) openDb() (err error, db *bbolt.DB) {
	db, err = bbolt.Open(s.cfg.StoragePath+"/ephemera.bbolt", 0600, &bbolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		err = fmt.Errorf("can't open database: %w", err)
		return
	}
	return
}

func (s *Storage) savePersistent(secret *Secret) error {
	err, db := s.openDb()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bbolt.Tx) error {
		bTime := tx.Bucket(TimeKeysName)
		bValue := tx.Bucket(ValueKeysName)
		id := RandString(s.cfg.IdLength)
		bId := []byte(id)
		for bValue.Get(bId) != nil {
			id = RandString(s.cfg.IdLength)
			bId = []byte(id)
		}
		secret.Id = id
		data, err := json.Marshal(secret)
		if err != nil {
			return fmt.Errorf("can't serialize data: %w", err)
		}
		err = bValue.Put(bId, data)
		if err != nil {
			return fmt.Errorf("can't save data to database: %w", err)
		}
		err = bTime.Put(secret.TimeKey(), bId)
		if err != nil {
			return fmt.Errorf("can't save time index: %w", err)
		}
		return nil
	})
}

func (s *Storage) saveLocal(secret *Secret) error {
	s.Lock()
	defer s.Unlock()
	id := RandString(s.cfg.IdLength)
	for _, ok := s.data[id]; ok; id = RandString(s.cfg.IdLength) {

	}
	secret.Id = id
	s.data[id] = secret
	return nil
}

func (s *Storage) GetSecret(id, key string) (error, string) {
	var err error
	var data string
	if s.cfg.PersistentStorage {
		err, data = s.getPersistent(id, key)
	} else {
		err, data = s.getLocal(id, key)
	}
	if err != nil {
		return err, ""
	}
	return nil, data
}

func (s *Storage) getPersistent(id, key string) (error, string) {
	err, db := s.openDb()
	if err != nil {
		return err, ""
	}
	var data string
	err = db.Update(func(tx *bbolt.Tx) error {
		bValue := tx.Bucket(ValueKeysName)
		v := bValue.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("secret not found")
		}
		sec := &Secret{}
		if err := json.Unmarshal(v, sec); err != nil {
			return err
		}
		err, data = Decrypt(s.cfg.KeyPart + key, sec.Secret)
		if err != nil {
			return err
		}
		bValue.Delete([]byte(id))
		tx.Bucket(TimeKeysName).Delete(sec.TimeKey())
		return nil
	})
	return err, data
}

func (s *Storage) getLocal(id, key string) (error, string) {
	s.Lock()
	defer s.Unlock()
	sec, ok := s.data[id]
	if !ok {
		return fmt.Errorf("secret not found"), ""
	}
	err, data := Decrypt(s.cfg.KeyPart + key, sec.Secret)
	if err != nil {
		return err, ""
	}
	delete(s.data, id)
	return nil, data
}

func (s *Storage) Clear() {
	s.cron.Clear()
}

func (s *Storage) clearExpired() {
	if s.cfg.PersistentStorage {
		s.clearExpiredPersistent()
	} else {
		s.clearExpiredLocal()
	}
}

func (s *Storage) clearExpiredLocal() {
	s.Lock()
	defer s.Unlock()
	expired := time.Now().Add(-24 * time.Hour)
	keysToDelete := make([]string, 0)
	for k, v := range s.data {
		if v.Created.Before(expired) {
			keysToDelete = append(keysToDelete, k)
		}
	}
	for _, k := range keysToDelete {
		delete(s.data, k)
	}
}

func (s *Storage) clearExpiredPersistent() {
	err, db := s.openDb()
	if err != nil {
		log.Printf("Can't open database for clearing: %v", err)
		return
	}
	defer db.Close()
	db.Update(func(tx *bbolt.Tx) error {
		bTime := tx.Bucket(TimeKeysName)
		c := bTime.Cursor()
		bValue := tx.Bucket(ValueKeysName)
		timeKeysToDelete := make([][]byte, 0)
		valueKeysToDelete := make([][]byte, 0)
		max := []byte(time.Now().Add(-24 * time.Hour).Format(time.RFC3339))
		for k, v := c.First(); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			timeKeysToDelete = append(timeKeysToDelete, k)
			valueKeysToDelete = append(valueKeysToDelete, v)
		}
		for _, k := range valueKeysToDelete {
			bValue.Delete(k)
		}
		for _, k := range timeKeysToDelete {
			bTime.Delete(k)
		}
		return nil
	})
}

