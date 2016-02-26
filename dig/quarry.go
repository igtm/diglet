package dig

import (
	_ "github.com/buckhx/diglet/geo"
	"github.com/buckhx/diglet/geo/osm"
	"github.com/buckhx/diglet/util"
	"sync"
)

type Quarry struct {
	db  *Qdb
	rdx *rIndex
}

func OpenQuarry(path string) (q *Quarry, err error) {
	db, err := OpenQdb(path)
	if err != nil {
		return
	}
	rdx := loadRIndex(db)
	q = &Quarry{
		db:  db,
		rdx: rdx,
	}
	return
}

func (q *Quarry) Dig(query Address) (match Address) {
	query = q.db.enrichPostcode(query)
	maxdist := 0.0
	for addr := range q.db.Search(query) {
		d := query.dist(addr)
		if d > maxdist {
			maxdist = d
			match = addr
		}
		util.Info("%s: %f", addr, d)
	}
	if maxdist == 0 {
		match = query
		match.HouseNumber = ""
		match.Street = ""
	} else {
		match.City = query.City
		match.Region = query.Region
		match.Country = query.Country
		match.Postcode = query.Postcode
	}
	return match
}

func (q *Quarry) Excavate(pbf, postcodes string) (err error) {
	util.Info("Excavating...")
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go q.survey(postcodes, wg)
	go q.excavate(pbf, 8, wg)
	wg.Wait()
	q.index()
	return
}

func (q *Quarry) index() {
	q.rdx = loadRIndex(q.db)
	q.db.indexAddresses(q.rdx)
}

func (q *Quarry) survey(postcode_path string, wg *sync.WaitGroup) {
	defer wg.Done()
	postcodes := ReadPostcodes(postcode_path)
	recs := make(chan QdbRecord)
	go func() {
		defer close(recs)
		for p := range postcodes {
			p.RelationKey = q.rdx.getRelation(p.Center)
			recs <- p
		}
	}()
	q.db.addRecords(PostcodeBucket, recs)
}

func (q *Quarry) excavate(pbf string, workers int, wg *sync.WaitGroup) {
	defer wg.Done()
	ex, err := osm.NewExcavator(pbf)
	util.Check(err)
	addrFilter := NewOsmFilter(1 << 27)
	ex.RelationCourier = func(feed <-chan *osm.Relation) {
		rels := make(chan QdbRecord)
		go func() {
			defer close(rels)
			for rel := range feed {
				if rel.IsSubregionBoundary() {
					for _, m := range rel.Members {
						if m.Type == osm.WayType {
							addrFilter.AddInt64(m.ID)
						}
						rels <- rel
					}
				}
			}
		}()
		q.db.addRecords(RelationBucket, rels)
	}
	err = ex.Start(workers)
	util.Check(err)
	ex, err = osm.NewExcavator(pbf)
	util.Check(err)
	ex.WayCourier = func(feed <-chan *osm.Way) {
		ways := make(chan QdbRecord)
		go func() {
			defer close(ways)
			for way := range feed {
				if way.IsAddressable() {
					addrFilter.AddInt64(way.ID)
					addrFilter.AddInt64(way.NodeIDs[0])
				}
				if addrFilter.HasInt64(way.ID) {
					for _, nid := range way.NodeIDs {
						addrFilter.AddInt64(nid)
					}
					ways <- way
				}
			}
		}()
		q.db.addRecords(WayBucket, ways)
	}
	ex.NodeCourier = func(feed <-chan *osm.Node) {
		for node := range feed {
			if node.IsAddressable() {
				addrFilter.AddInt64(node.ID)
			}
		}
	}
	err = ex.Start(workers)
	util.Check(err)
	ex, err = osm.NewExcavator(pbf)
	util.Check(err)
	ex.NodeCourier = func(feed <-chan *osm.Node) {
		nodes := make(chan QdbRecord)
		go func() {
			defer close(nodes)
			for node := range feed {
				if addrFilter.HasInt64(node.ID) {
					nodes <- node
				}
			}
		}()
		q.db.addRecords(NodeBucket, nodes)
	}
	err = ex.Start(1)
	util.Check(err)
}

func addressRelations(q *Qdb, addr Address) <-chan string {
	keys := make(chan string)
	go func() {
		defer close(keys)
		for pc := range q.Postcodes(addr.Country, addr.Postcode) {
			keys <- pc.RelationKey
		}
	}()
	return keys
}
