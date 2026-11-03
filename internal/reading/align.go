package reading

// align uses minimum word edits, with stable ties and narrowly defined zero-cost
// spoken equivalences. Function words stay as anchors and are filtered AFTER
// alignment, so removing 'is/to' cannot shift all later words.
func align(a, b []token) []Difference {
	n, m := len(a), len(b)
	width := m + 1
	cost := make([]uint16, (n+1)*width)
	back := make([]uint8, len(cost))
	for i := 1; i <= n; i++ {
		cost[i*width] = uint16(i)
		back[i*width] = 3
	}
	for j := 1; j <= m; j++ {
		cost[j] = uint16(j)
		back[j] = 4
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			k := i*width + j
			value := cost[(i-1)*width+j-1]
			action := uint8(1)
			if !equivalentWord(a[i-1].word, b[j-1].word) {
				value++
				action = 2
			}
			// Spoken "it's" and ASR "its" are equivalent spelling evidence.
			// A separate extra 'it' still remains an insertion (robot is / robot it's).
			if i >= 2 && a[i-2].word == "it" && a[i-1].word == "is" && b[j-1].word == "its" {
				v := cost[(i-2)*width+j-1]
				if v <= value {
					value = v
					action = 5
				}
			}
			if j >= 2 && b[j-2].word == "it" && b[j-1].word == "is" && a[i-1].word == "its" {
				v := cost[(i-1)*width+j-2]
				if v <= value {
					value = v
					action = 6
				}
			}
			if v := cost[(i-1)*width+j] + 1; v < value {
				value = v
				action = 3
			}
			if v := cost[i*width+j-1] + 1; v < value {
				value = v
				action = 4
			}
			cost[k] = value
			back[k] = action
		}
	}
	out := []Difference{}
	appendRef := func(kind string, x token, hyp string) {
		out = append(out, Difference{Kind: kind, Reference: x.word, Transcript: hyp, ReferencePosition: x.position, Ignored: functionWords[x.word]})
	}
	for i, j := n, m; i > 0 || j > 0; {
		switch back[i*width+j] {
		case 1:
			kind := "match"
			if a[i-1].word != b[j-1].word {
				kind = "equivalent"
			}
			appendRef(kind, a[i-1], b[j-1].word)
			i--
			j--
		case 2:
			appendRef("substitute", a[i-1], b[j-1].word)
			i--
			j--
		case 3:
			appendRef("delete", a[i-1], "")
			i--
		case 4:
			out = append(out, Difference{Kind: "insert", Transcript: b[j-1].word, ReferencePosition: -1, Ignored: functionWords[b[j-1].word]})
			j--
		case 5:
			appendRef("equivalent", a[i-1], "its")
			appendRef("equivalent", a[i-2], "its")
			i -= 2
			j--
		case 6:
			appendRef("equivalent", a[i-1], "it is")
			i--
			j -= 2
		default:
			panic("invalid reading alignment")
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
