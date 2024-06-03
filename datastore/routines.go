package datastore

type HashOperation struct {
	put   bool
	key   string
	index int64
}

type SegmentPosition struct {
	segment  *Segment
	position int64
}

type EntryElement struct {
	ent entry
	err chan error
}

type HashOperator struct {
	queries chan HashOperation
	answers chan *SegmentPosition
}

func (db *Db) handleInput() {
	go func() {
		for {
			ee := <-db.ops
			stat, err := db.out.Stat()
			if err != nil {
				ee.err <- err
				continue
			}

			if stat.Size()+ee.ent.Length() > db.segments.size {
				err := db.addSegment()
				if err != nil {
					ee.err <- err
					continue
				}
			}

			n, err := db.out.Write(ee.ent.Encode())
			if err == nil {
				db.operator.queries <- HashOperation{
					put:   true,
					key:   ee.ent.key,
					index: int64(n),
				}
			}
			ee.err <- nil
		}
	}()
}

func (db *Db) handleOperations() {
	go func() {
		for {
			op := <-db.operator.queries
			if op.put {
				db.segments.GetLast().index[op.key] = db.offset
				db.offset += op.index
				continue
			}

			seg, pos, err := db.segments.Find(op.key)
			if err != nil {
				db.operator.answers <- nil
				continue
			}

			db.operator.answers <- &SegmentPosition{
				seg,
				pos,
			}
		}
	}()
}
