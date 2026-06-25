package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"go.etcd.io/bbolt"
)

type Secret struct {
	ID      string
	Secret  []byte
	Created time.Time
}

func (s *Secret) TimeKey() []byte {
	return []byte(s.Created.Format(time.RFC3339) + s.ID)
}

type Storage struct {
	cfg  *Config
	data map[string]*Secret
	cron *gocron.Scheduler
	sync.RWMutex
}

var (
	TimeKeysName  = []byte("timekeys")
	ValueKeysName = []byte("valuekeys")
)

func NewStorage(cfg *Config) *Storage {
	st := &Storage{cfg: cfg, data: make(map[string]*Secret), cron: gocron.NewScheduler(time.UTC)}
	var err error
	_, err = st.cron.Every(cfg.RunClearingInterval).Seconds().Do(st.clearExpired)
	if err != nil {
		log.Fatalf("Can't create cronjob: %v", err)
	}
	st.cron.StartAsync()
	if cfg.PersistentStorage {
		_ = os.Mkdir(cfg.StoragePath, 0700)
		db, err := st.openDB()
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

func (s *Storage) SaveSecret(secret string) (id, key string, err error) {
	key = RandString(s.cfg.KeyLength)
	data, err := Encrypt(s.cfg.KeyPart+key, secret)
	if err != nil {
		return
	}
	sec := &Secret{
		Secret:  data,
		Created: time.Now(),
	}
	if s.cfg.PersistentStorage {
		err = s.savePersistent(sec)
	} else {
		err = s.saveLocal(sec)
	}
	id = sec.ID
	return
}

func (s *Storage) openDB() (db *bbolt.DB, err error) {
	db, err = bbolt.Open(s.cfg.StoragePath+"/ephemera.bbolt", 0600, &bbolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		err = fmt.Errorf("can't open database: %w", err)
		return
	}
	return
}

func (s *Storage) savePersistent(secret *Secret) error {
	db, err := s.openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bbolt.Tx) error {
		bTime := tx.Bucket(TimeKeysName)
		bValue := tx.Bucket(ValueKeysName)
		id := RandString(s.cfg.IDLength)
		bID := []byte(id)
		for bValue.Get(bID) != nil {
			id = RandString(s.cfg.IDLength)
			bID = []byte(id)
		}
		secret.ID = id
		data, err := json.Marshal(secret)
		if err != nil {
			return fmt.Errorf("can't serialize data: %w", err)
		}
		err = bValue.Put(bID, data)
		if err != nil {
			return fmt.Errorf("can't save data to database: %w", err)
		}
		err = bTime.Put(secret.TimeKey(), bID)
		if err != nil {
			return fmt.Errorf("can't save time index: %w", err)
		}
		return nil
	})
}

func (s *Storage) saveLocal(secret *Secret) error {
	s.Lock()
	defer s.Unlock()
	id := RandString(s.cfg.IDLength)
	for _, ok := s.data[id]; ok; id = RandString(s.cfg.IDLength) {
	}
	secret.ID = id
	s.data[id] = secret
	return nil
}

func (s *Storage) GetSecret(id, key string) (string, error) {
	var err error
	var data string
	if s.cfg.PersistentStorage {
		data, err = s.getPersistent(id, key)
	} else {
		data, err = s.getLocal(id, key)
	}
	if err != nil {
		return "", err
	}
	return data, nil
}

func (s *Storage) getPersistent(id, key string) (string, error) {
	db, err := s.openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()
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
		data, err = Decrypt(s.cfg.KeyPart+key, sec.Secret)
		if err != nil {
			return err
		}
		bValue.Delete([]byte(id))
		tx.Bucket(TimeKeysName).Delete(sec.TimeKey())
		return nil
	})
	return data, err
}

func (s *Storage) getLocal(id, key string) (string, error) {
	s.Lock()
	defer s.Unlock()
	sec, ok := s.data[id]
	if !ok {
		return "", fmt.Errorf("secret not found")
	}
	data, err := Decrypt(s.cfg.KeyPart+key, sec.Secret)
	if err != nil {
		return "", err
	}
	delete(s.data, id)
	return data, nil
}

func (s *Storage) Clear() {
	s.cron.Clear()
}

func (s *Storage) clearExpired() {
	log.Println("Run clearing")
	if s.cfg.PersistentStorage {
		s.clearExpiredPersistent()
	} else {
		s.clearExpiredLocal()
	}
}

func (s *Storage) clearExpiredLocal() {
	s.Lock()
	defer s.Unlock()
	expired := time.Now().Add(time.Duration(-1*s.cfg.SecretsExpire) * time.Second)
	keysToDelete := make([]string, 0)
	for k, v := range s.data {
		if v.Created.Before(expired) {
			keysToDelete = append(keysToDelete, k)
		}
	}
	log.Printf("Found %d expired secrets", len(keysToDelete))
	for _, k := range keysToDelete {
		delete(s.data, k)
	}
}

func (s *Storage) clearExpiredPersistent() {
	db, err := s.openDB()
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
		max := []byte(time.Now().Add(time.Duration(-1*s.cfg.SecretsExpire) * time.Second).Format(time.RFC3339))
		for k, v := c.First(); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			timeKeysToDelete = append(timeKeysToDelete, k)
			valueKeysToDelete = append(valueKeysToDelete, v)
		}
		log.Printf("Found %d expired secrets", len(timeKeysToDelete))
		for _, k := range valueKeysToDelete {
			bValue.Delete(k)
		}
		for _, k := range timeKeysToDelete {
			bTime.Delete(k)
		}
		return nil
	})
}
