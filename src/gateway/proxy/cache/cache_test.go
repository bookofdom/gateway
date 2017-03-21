package proxy

import (
	"gateway/model"
	"testing"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func TestCache(t *testing.T) { gc.TestingT(t) }

type CacheSuite struct{}

var _ = gc.Suite(&CacheSuite{})

type cacherMock struct {
	cache map[interface{}]interface{}
}

func (c *cacherMock) Purge() {
	c.cache = make(map[interface{}]interface{})
}

func (c *cacherMock) Get(key interface{}) (interface{}, bool) {
	if val, ok := c.cache[key]; ok {
		return val, ok
	}
	return nil, false
}

func (c *cacherMock) Add(key, value interface{}) bool {
	c.cache[key] = value
	return true
}

func (c *cacherMock) Remove(key interface{}) bool {
	delete(c.cache, key)
	return true
}

func (c *cacherMock) Contains(key interface{}) bool {
	if _, exists := c.cache[key]; exists {
		return true
	}
	return false
}

func (c *cacherMock) Len() int {
	return len(c.cache)
}

func newCacherMock() *cacherMock {
	c := make(map[interface{}]interface{})
	return &cacherMock{c}
}

type dataSourceMock struct{}

func (d *dataSourceMock) FindProxyEndpointForProxy(id int64, endpointType string) (*model.ProxyEndpoint, error) {
	endpoint := &model.ProxyEndpoint{}
	endpoint.ID = id
	endpoint.APIID = 1
	return endpoint, nil
}

func (d *dataSourceMock) AllLibrariesForProxy(apiid int64) ([]*model.Library, error) {
	ids := []int64{1, 2, 3}
	libraries := make([]*model.Library, len(ids))
	for i, t := range ids {
		library := &model.Library{}
		library.APIID = apiid
		library.ID = t
		libraries[i] = library
	}
	return libraries, nil
}

func (d *dataSourceMock) FindPlanByAccountID(accountid int64) (*model.Plan, error) {
	plan := &model.Plan{}
	plan.ID = 1
	return plan, nil
}

func (d *dataSourceMock) FindHostForHostname(hostname string) (*model.Host, error) {
	host := &model.Host{}
	host.ID = 1
	host.APIID = 1
	host.Hostname = hostname
	return host, nil
}

func (s *CacheSuite) TestCaches(c *gc.C) {
	for i, t := range []struct {
		should              string
		dataSourceCacheType interface{}
		criteria            CacheCriteria
		badCriteria         CacheCriteria
	}{{
		should:              "create a functioning endpointCache",
		dataSourceCacheType: endpointCache{},
		criteria:            int64(1),
		badCriteria:         "foobar",
	}, {
		should:              "create a functioning libraryCache",
		dataSourceCacheType: libraryCache{},
		criteria:            int64(1),
		badCriteria:         "foobar",
	}, {
		should:              "create a functioning planCache",
		dataSourceCacheType: planCache{},
		criteria:            int64(1),
		badCriteria:         "foobar",
	}, {
		should:              "create a functioning hostCache",
		dataSourceCacheType: hostCache{},
		criteria:            "foo",
		badCriteria:         int64(1),
	}} {
		c.Logf("Test %d: should %s", i, t.should)
		cm := newCacherMock()

		var cache DataSourceCache
		switch t.dataSourceCacheType.(type) {
		case endpointCache:
			cache = newEndpointCache(&dataSourceMock{}, cm)
		case libraryCache:
			cache = newLibraryCache(&dataSourceMock{}, cm)
		case planCache:
			cache = newPlanCache(&dataSourceMock{}, cm)
		case hostCache:
			cache = newHostCache(&dataSourceMock{}, cm)
		default:
			c.Fatal("invalid cache type")
		}

		// Get should return the model and add it to the cache
		val, err := cache.Get(t.criteria)
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(val, gc.NotNil)

		// Cache should now contain the model
		c.Assert(cm.Len(), gc.Equals, 1)
		c.Assert(cm.Contains(t.criteria), jc.IsTrue)

		// Should return the same model
		again, err := cache.Get(t.criteria)
		c.Assert(again, gc.DeepEquals, val)

		// Cache should still have 1 entry
		c.Assert(cm.Len(), gc.Equals, 1)

		// Cache should contain entry
		c.Assert(cache.Contains(t.criteria), jc.IsTrue)

		// Remove the entry from the cache
		removed := cache.Remove(t.criteria)
		c.Assert(removed, jc.IsTrue)

		// Cache should be empty
		c.Assert(cm.Len(), gc.Equals, 0)

		// Cache should not contain entry
		c.Assert(cache.Contains(t.criteria), jc.IsFalse)

		// Incorrect criteria type should return an error and not cause a panic
		val, err = cache.Get(t.badCriteria)
		c.Assert(val, gc.IsNil)
		c.Assert(err, gc.NotNil)
	}
}
