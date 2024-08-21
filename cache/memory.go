package cache

import (
	"encoding/json"

	"github.com/echovault/echovault/echovault"
)

type CacheService struct {
	cache *echovault.EchoVault
}

func NewCacheService() (*CacheService, error) {
	server, err := echovault.NewEchoVault()
	if err != nil {
		return nil, err
	}

	c := &CacheService{
		cache: server,
	}

	return c, nil
}

func (c *CacheService) Get(key string) (map[string]interface{}, error) {
	value, err := c.cache.Get(key)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(value), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *CacheService) Set(key string, data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, _, err = c.cache.Set(key, string(jsonData), echovault.SetOptions{})
	if err != nil {
		return err
	}

	return nil
}
