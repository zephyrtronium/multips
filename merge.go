package main

func Conflict(p1, p2 Patch) (c1, c2 []Write) {
	i, j := 0, 0
	w1, w2 := p1[0], p2[0]
	org1, org2 := w1.Org(), w2.Org()
	end1, end2 := org1+w1.Len(), org2+w2.Len()
	for {
		if org1 >= org2 && org1 < end2 || org2 >= org1 && org2 < end1 {
			c1 = append(c1, w1)
			c2 = append(c2, w2)
		}
		if end1 < end2 {
			i++
			if i >= len(p1) {
				return
			}
			w1 = p1[i]
			org1 = w1.Org()
			end1 = org1 + w1.Len()
		} else if end1 > end2 {
			j++
			if j >= len(p2) {
				return
			}
			w2 = p2[j]
			org2 = w2.Org()
			end2 = org2 + w2.Len()
		} else {
			i++
			j++
			if i >= len(p1) || j >= len(p2) {
				return
			}
			w1, w2 = p1[i], p2[j]
			org1, org2 = w1.Org(), w2.Org()
			end1, end2 = org1+w1.Len(), org2+w2.Len()
		}
	}
	panic("unreachable")
}

func Merge(p1, p2 Patch) (merged Patch) {
	i, j := 0, 0
	for {
		if p1[i].Org() < p2[j].Org() {
			merged = append(merged, p1[i])
			if i++; i >= len(p1) {
				merged = append(merged, p2[j:]...)
				return
			}
		} else {
			merged = append(merged, p2[j])
			if j++; j >= len(p2) {
				merged = append(merged, p1[i:]...)
				return
			}
		}
	}
	panic("unreachable")
}
