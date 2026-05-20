package doc

import (
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// SortDocument recursively sorts all map/documents by key so that
// serialization produces deterministic output regardless of Go map
// iteration order.
func SortDocument(v interface{}) interface{} {
	switch val := v.(type) {
	case bson.M:
		return sortMap(val)
	case bson.D:
		return sortD(val)
	case bson.A:
		return sortA(val)
	default:
		return v
	}
}

func sortMap(m bson.M) bson.D {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(bson.D, 0, len(m))
	for _, k := range keys {
		out = append(out, bson.E{Key: k, Value: SortDocument(m[k])})
	}
	return out
}

func sortD(d bson.D) bson.D {
	out := make(bson.D, 0, len(d))
	for _, e := range d {
		out = append(out, bson.E{Key: e.Key, Value: SortDocument(e.Value)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func sortA(a bson.A) bson.A {
	out := make(bson.A, len(a))
	for i, v := range a {
		out[i] = SortDocument(v)
	}
	return out
}
