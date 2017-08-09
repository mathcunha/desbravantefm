package main

import "time"

type cache struct {
	feeds map[string]*rss
}

//NewCache builds a new cache using the columns as key and nil as value
func NewCache(columns ...string) (c *cache) {
	c = &cache{feeds: make(map[string]*rss)}
	for _, v := range columns {
		c.feeds[v] = nil
	}
	tick := time.Tick(60 * time.Minute)
	go func() {
		c.Update()
		for e := range tick {
			logger.Printf("Updating feeds at %v\n", e)
			c.Update()
		}
	}()
	return
}

func (c *cache) Update() {
	for col := range c.feeds {
		rss := &rss{}
		if err := rss.load(col); err == nil {
			c.feeds[col] = rss
			logger.Printf("%v Updated \n", col)
		} else {
			logger.Printf("%v not updated: %v", col, err)
		}
		//avoid beeing accused of DDoS
		time.Sleep(10 * time.Second)
	}
}

func (c *cache) Get(col string) (*rss, bool) {
	rss, has := c.feeds[col]
	return rss, has
}

func (c *cache) Set(col string, rss *rss) {
	c.feeds[col] = rss
}
