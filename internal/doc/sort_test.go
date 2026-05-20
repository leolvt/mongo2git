package doc

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestSortDocument_SortsTopLevelKeys(t *testing.T) {
	in := bson.M{"z": 1, "a": 2, "m": 3}
	got := SortDocument(in)

	d, ok := got.(bson.D)
	if !ok {
		t.Fatalf("expected bson.D, got %T", got)
	}
	want := bson.D{
		{Key: "a", Value: 2},
		{Key: "m", Value: 3},
		{Key: "z", Value: 1},
	}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("got %v, want %v", d, want)
	}
}

func TestSortDocument_RecursiveNestedMap(t *testing.T) {
	in := bson.M{
		"outer": bson.M{"b": 2, "a": 1},
		"alpha": 1,
	}
	got := SortDocument(in)

	d, ok := got.(bson.D)
	if !ok {
		t.Fatalf("expected bson.D, got %T", got)
	}
	// Outer keys sorted: alpha, outer
	if d[0].Key != "alpha" || d[1].Key != "outer" {
		t.Fatalf("outer keys not sorted: %v", d)
	}
	inner, ok := d[1].Value.(bson.D)
	if !ok {
		t.Fatalf("expected nested bson.D, got %T", d[1].Value)
	}
	wantInner := bson.D{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	if !reflect.DeepEqual(inner, wantInner) {
		t.Errorf("nested: got %v, want %v", inner, wantInner)
	}
}

func TestSortDocument_ArrayOfMaps(t *testing.T) {
	in := bson.M{
		"_id": "arr-test",
		"items": bson.A{
			bson.M{"b": 2, "a": 1},
			"scalar",
		},
	}
	got := SortDocument(in)

	d := got.(bson.D)
	arr := d[1].Value.(bson.A)
	sortedMap := arr[0].(bson.D)
	want := bson.D{{Key: "a", Value: 1}, {Key: "b", Value: 2}}
	if !reflect.DeepEqual(sortedMap, want) {
		t.Errorf("array element: got %v, want %v", sortedMap, want)
	}
	if arr[1] != "scalar" {
		t.Errorf("scalar in array changed: got %v", arr[1])
	}
}

func TestSortDocument_SortsD(t *testing.T) {
	in := bson.D{{Key: "z", Value: 3}, {Key: "a", Value: 1}}
	got := SortDocument(in)

	d := got.(bson.D)
	want := bson.D{{Key: "a", Value: 1}, {Key: "z", Value: 3}}
	if !reflect.DeepEqual(d, want) {
		t.Errorf("got %v, want %v", d, want)
	}
}

func TestSortDocument_EmptyMap(t *testing.T) {
	in := bson.M{}
	got := SortDocument(in)

	d := got.(bson.D)
	if len(d) != 0 {
		t.Errorf("expected empty D, got %v", d)
	}
}

func TestSortDocument_ScalarPassthrough(t *testing.T) {
	if got := SortDocument("hello"); got != "hello" {
		t.Errorf("string changed: got %v", got)
	}
	if got := SortDocument(42); got != 42 {
		t.Errorf("int changed: got %v", got)
	}
	if got := SortDocument(true); got != true {
		t.Errorf("bool changed: got %v", got)
	}
	id := bson.NewObjectID()
	if got := SortDocument(id); got != id {
		t.Errorf("ObjectID changed: got %v", got)
	}
}

func TestSortDocument_Idempotent(t *testing.T) {
	in := bson.M{"z": bson.M{"b": 2, "a": 1}, "a": 1}
	pass1 := SortDocument(in)
	pass2 := SortDocument(pass1)
	if !reflect.DeepEqual(pass1, pass2) {
		t.Errorf("not idempotent:\n  pass1: %v\n  pass2: %v", pass1, pass2)
	}
}

func TestSortDocument_DeeplyNested(t *testing.T) {
	in := bson.M{
		"a": bson.A{
			bson.M{"nested": bson.M{"deep": bson.M{"c": 3, "b": 2, "a": 1}}},
		},
	}
	got := SortDocument(in)
	d := got.(bson.D)
	arr := d[0].Value.(bson.A)
	outerMap := arr[0].(bson.D)
	innerMap := outerMap[0].Value.(bson.D)
	deepMap := innerMap[0].Value.(bson.D)

	want := bson.D{{Key: "a", Value: 1}, {Key: "b", Value: 2}, {Key: "c", Value: 3}}
	if !reflect.DeepEqual(deepMap, want) {
		t.Errorf("deeply nested: got %v, want %v", deepMap, want)
	}
}
