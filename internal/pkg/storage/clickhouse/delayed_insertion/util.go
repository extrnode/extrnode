package delayed_insertion

func (c *Collector[collectorPossibleTypes]) Add(s collectorPossibleTypes) {
	if c.chStorage == nil {
		return
	}

	c.mx.Lock()
	c.cache = append(c.cache, s)
	c.mx.Unlock()
}
func (c *Collector[collectorPossibleTypes]) getCachedEntries() (s []collectorPossibleTypes) {
	c.mx.Lock()
	s = c.cache
	c.cache = make([]collectorPossibleTypes, 0, flushAmount)
	c.mx.Unlock()

	return
}
