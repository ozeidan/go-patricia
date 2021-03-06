// Copyright (c) 2014 The go-patricia AUTHORS
//
// Use of this source code is governed by The MIT License
// that can be found in the LICENSE file.

package patricia

import (
	"crypto/rand"
	mrand "math/rand"
	"reflect"
	"testing"
)

// Tests -----------------------------------------------------------------------

func TestTrie_GetNonexistentPrefix(t *testing.T) {
	trie := NewTrie()

	data := []testData{
		{"aba", 0, success},
	}

	for _, v := range data {
		t.Logf("INSERT prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
		if ok := trie.Insert(Prefix(v.key), v.value); ok != v.retVal {
			t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
		}
	}

	t.Logf("GET prefix=baa, expect item=nil")
	if item := trie.Get(Prefix("baa")); item != nil {
		t.Errorf("Unexpected return value, expected=<nil>, got=%v", item)
	}
}

func TestTrie_RandomKitchenSink(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	const count, size = 750000, 16
	b := make([]byte, count+size+1)
	if _, err := rand.Read(b); err != nil {
		t.Fatal("error generating random bytes", err)
	}
	m := make(map[string]string)
	for i := 0; i < count; i++ {
		m[string(b[i:i+size])] = string(b[i+1 : i+size+1])
	}
	trie := NewTrie()
	getAndDelete := func(k, v string) {
		i := trie.Get(Prefix(k))
		if i == nil {
			t.Fatalf("item not found, prefix=%v", []byte(k))
		} else if s, ok := i.(string); !ok {
			t.Fatalf("unexpected item type, expecting=%v, got=%v", reflect.TypeOf(k), reflect.TypeOf(i))
		} else if s != v {
			t.Fatalf("unexpected item, expecting=%v, got=%v", []byte(k), []byte(s))
		} else if !trie.Delete(Prefix(k)) {
			t.Fatalf("delete failed, prefix=%v", []byte(k))
		} else if i = trie.Get(Prefix(k)); i != nil {
			t.Fatalf("unexpected item, expecting=<nil>, got=%v", i)
		} else if trie.Delete(Prefix(k)) {
			t.Fatalf("extra delete succeeded, prefix=%v", []byte(k))
		}
	}
	for k, v := range m {
		if !trie.Insert(Prefix(k), v) {
			t.Fatalf("insert failed, prefix=%v", []byte(k))
		}
		if byte(k[size/2]) < 128 {
			getAndDelete(k, v)
			delete(m, k)
		}
	}
	for k, v := range m {
		getAndDelete(k, v)
	}
}

// Make sure Delete that affects the root node works.
// This was panicking when Delete was broken.
func TestTrie_DeleteRoot(t *testing.T) {
	trie := NewTrie()

	v := testData{"aba", 0, success}

	t.Logf("INSERT prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
	if ok := trie.Insert(Prefix(v.key), v.value); ok != v.retVal {
		t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
	}

	t.Logf("DELETE prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
	if ok := trie.Delete(Prefix(v.key)); ok != v.retVal {
		t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
	}
}

func TestTrie_DeleteAbsentPrefix(t *testing.T) {
	trie := NewTrie()

	v := testData{"a", 0, success}

	t.Logf("INSERT prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
	if ok := trie.Insert(Prefix(v.key), v.value); ok != v.retVal {
		t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
	}

	d := "ab"
	t.Logf("DELETE prefix=%v, success=%v", d, failure)
	if ok := trie.Delete(Prefix(d)); ok != failure {
		t.Errorf("Unexpected return value, expected=%v, got=%v", failure, ok)
	}
	t.Logf("GET prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
	if i := trie.Get(Prefix(v.key)); i != v.value {
		t.Errorf("Unexpected item, expected=%v, got=%v", v.value, i)
	}
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func checkMasksRecursive(t *testing.T, root *Trie) {
	for _, child := range root.children.getChildren() {
		if child.mask & ^root.mask != 0 {
			t.Errorf("\ninvalid mask at prefix %s\nchild prefix: %s\ncharmap: \t%s\nmask: \t%064b\n"+
				"child mask: \t%064b\ndiff:\t%064b\n",
				root.prefix,
				child.prefix,
				reverse(charmap),
				root.mask,
				child.mask,
				child.mask & ^root.mask,
			)
		}
		checkMasksRecursive(t, child)
	}
}

func TestTrie_AddCorrectMasks(t *testing.T) {
	trie := NewTrie()
	data := []testData{
		{"Pepan", "Pepan Zdepan", success},
		{"Pepin", "Pepin Omacka", success},
		{"Honza", "Honza Novak", success},
		{"Jenik", "Jenik Poustevnicek", success},
		{"Pepan", "Pepan Dupan", failure},
		{"Karel", "Karel Pekar", success},
		{"Jenak", "Jenak Poustevnicek", success},
		{"Pepanek", "Pepanek Zemlicka", success},
	}

	for _, v := range data {
		t.Logf("INSERT prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
		if ok := trie.Insert(Prefix(v.key), v.value); ok != v.retVal {
			t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
		}
		checkMasksRecursive(t, trie)
	}
}

func TestTrie_DeleteCorrectMasks(t *testing.T) {
	data := []testData{
		{"Pepan", "Pepan Zdepan", success},
		{"Pepin", "Pepin Omacka", success},
		{"Honza", "Honza Novak", success},
		{"Jenik", "Jenik Poustevnicek", success},
		{"Karel", "Karel Pekar", success},
		{"Jenak", "Jenak Poustevnicek", success},
		{"Pepanek", "Pepanek Zemlicka", success},
	}

	deleteData := [][]testData{
		{
			{"Honza", "Honza Novak", success},
			{"Jenik", "Jenik Poustevnicek", success},
			{"Pepan", "Pepan Dupan", success},
		},
		{
			{"Pepan", "Pepan Dupan", success},
		},
		{
			{"Jenak", "Jenak Poustevnicek", success},
			{"Pepanek", "Pepanek Zemlicka", success},
			{"Pepin", "Pepin Omacka", success},
			{"Honza", "Honza Novak", success},
			{"Jenik", "Jenik Poustevnicek", success},
		},
	}

	for _, d := range deleteData {
		trie := NewTrie()
		for _, v := range data {
			t.Logf("INSERT prefix=%v, item=%v, success=%v", v.key, v.value, v.retVal)
			if ok := trie.Insert(Prefix(v.key), v.value); ok != v.retVal {
				t.Errorf("Unexpected return value, expected=%v, got=%v", v.retVal, ok)
			}
		}

		for _, record := range d {
			trie.Delete(Prefix(record.key))
		}

		checkMasksRecursive(t, trie)
	}

}

func populateTrie(t *testing.T) *Trie {
	data := []string{
		"Pepan",
		"Pepin",
		"Honza",
		"Jenik",
		"Karel",
		"Jenak",
		"Pepanek",
	}

	trie := NewTrie()
	for _, v := range data {
		if ok := trie.Insert(Prefix(v), struct{}{}); !ok {
			t.Errorf("Couldn't insert item %s", v)
		}
	}

	return trie
}

func TestTrie_FuzzyCollect(t *testing.T) {
	trie := populateTrie(t)

	type testResult struct {
		wantKey     string
		wantSkipped int
	}

	type testData struct {
		query           string
		caseInsensitive bool
		wantResults     []testResult
	}

	testQueries := []testData{
		{
			"Ppn",
			false,
			[]testResult{
				{"Pepan", 2},
				{"Pepin", 2},
				{"Pepanek", 2},
			},
		},
		{
			"Ha",
			false,
			[]testResult{
				{"Honza", 3},
			},
		},
		{
			"nza",
			false,
			[]testResult{
				{"Honza", 0},
			},
		},
		{
			"eni",
			false,
			[]testResult{
				{"Jenik", 0},
			},
		},
		{
			"jk",
			true,
			[]testResult{
				{"Jenik", 3},
				{"Jenak", 3},
			},
		},
		{
			"ppn",
			true,
			[]testResult{
				{"Pepan", 2},
				{"Pepin", 2},
				{"Pepanek", 2},
			},
		},
	}

	for _, data := range testQueries {
		resultMap := make(map[string]int)
		t.Logf("QUERY %s", data.query)
		trie.VisitFuzzy(Prefix(data.query), data.caseInsensitive, func(prefix Prefix, item Item, skipped int) error {
			// result := testResult{string(prefix), skipped}
			resultMap[string(prefix)] = skipped
			return nil
		})
		t.Logf("got result set %v\n", resultMap)

		for _, want := range data.wantResults {
			got, ok := resultMap[want.wantKey]
			if !ok {
				t.Errorf("item %s not found in result set\n", want.wantKey)
				continue
			}

			if got != want.wantSkipped {
				t.Errorf("got wrong skipped value, wanted %d, got %d\n",
					want.wantSkipped, got)
			}
		}
	}
}

func TestTrie_SubstringCollect(t *testing.T) {
	trie := populateTrie(t)

	type testData struct {
		query           string
		caseInsensitive bool
		wantResults     []string
	}

	testQueries := []testData{
		{
			"epa",
			false,
			[]string{
				"Pepan",
				"Pepanek",
			},
		},
		{
			"onza",
			false,
			[]string{
				"Honza",
			},
		},
		{
			"nza",
			false,
			[]string{
				"Honza",
			},
		},
		{
			"l",
			false,
			[]string{
				"Karel",
			},
		},
		{
			"a",
			false,
			[]string{
				"Pepan",
				"Honza",
				"Pepan",
				"Karel",
				"Jenak",
				"Pepanek",
			},
		},
		{
			"pep",
			true,
			[]string{
				"Pepin",
				"Pepan",
			},
		},
		{
			"kar",
			true,
			[]string{
				"Karel",
			},
		},
		{
			"",
			false,
			[]string{
				"Pepan",
				"Pepin",
				"Honza",
				"Jenik",
				"Karel",
				"Jenak",
				"Pepanek",
			},
		},
	}

	for _, data := range testQueries {
		resultMap := make(map[string]bool)
		t.Logf("QUERY %s", data.query)
		trie.VisitSubstring(Prefix(data.query), true, func(prefix Prefix, item Item) error {
			// result := testResult{string(prefix), skipped}
			resultMap[string(prefix)] = true
			return nil
		})
		t.Logf("got result set %v\n", resultMap)

		for _, want := range data.wantResults {
			if _, ok := resultMap[want]; !ok {
				t.Errorf("item %s not found in result set\n", want)
				continue
			}
		}
	}
}

func Test_makePrefixMask(t *testing.T) {
	type testData struct {
		key    Prefix
		wanted uint64
	}

	data := []testData{
		{
			Prefix("0123456789"),
			0x3FF,
		},
		{
			Prefix("AAAA"),
			0x400,
		},
		{
			Prefix(""),
			0,
		},
		{
			Prefix("abc"),
			0x7000000000,
		},
		{
			Prefix(".-"),
			0xc000000000000000,
		},
	}

	for _, d := range data {
		got := makePrefixMask(d.key)
		if got != d.wanted {
			t.Errorf("Unexpected bitmask, wanted: %b, got %b\n", d.wanted, got)
		}
	}
}

const (
	amountWords = 100000
	wordLength  = 10
	queryLength = 10
)

var benchmarkTrie *Trie

func populateBenchmarkTrie(superDenseChildList bool) {
	benchmarkTrie = NewTrie()

	for i := 0; i < amountWords; i++ {
		benchmarkTrie.Insert(Prefix(mrandBytes(wordLength)), struct{}{})
	}
}

type visitFunc func(prefix Prefix, caseInsensitive bool, visitor VisitorFunc) error

func benchmarkVisit(caseInsensitive bool, visitor visitFunc, b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		visitor(Prefix(mrandBytes(queryLength)), caseInsensitive, func(prefix Prefix, item Item) error {
			return nil
		})
	}
}

func benchmarkVisitFuzzy(caseInsensitive bool, b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchmarkTrie.VisitFuzzy(Prefix(mrandBytes(queryLength)), caseInsensitive, func(prefix Prefix, item Item, skipped int) error {
			return nil
		})
	}
}

func BenchmarkPrefix(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisit(false, benchmarkTrie.VisitPrefixes, b)
}
func BenchmarkPrefixCaseInsensitive(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisit(true, benchmarkTrie.VisitPrefixes, b)
}
func BenchmarkPrefixSuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisit(false, benchmarkTrie.VisitPrefixes, b)
}
func BenchmarkPrefixCaseInsensitiveSuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisit(true, benchmarkTrie.VisitPrefixes, b)
}
func BenchmarkSubstring(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisit(false, benchmarkTrie.VisitSubstring, b)
}
func BenchmarkSubstringCaseInsensitive(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisit(true, benchmarkTrie.VisitSubstring, b)
}
func BenchmarkSubstringSuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisit(false, benchmarkTrie.VisitSubstring, b)
}
func BenchmarkSubstringCaseInsensitiveSuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisit(true, benchmarkTrie.VisitSubstring, b)
}

func BenchmarkFuzzy(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisitFuzzy(false, b)
}
func BenchmarkFuzzyCaseInsensitive(b *testing.B) {
	populateBenchmarkTrie(false)
	benchmarkVisitFuzzy(true, b)
}
func BenchmarkFuzzySuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisitFuzzy(false, b)
}
func BenchmarkFuzzyCaseInsensitiveSuperDense(b *testing.B) {
	populateBenchmarkTrie(true)
	benchmarkVisitFuzzy(true, b)
}

func mrandBytes(length int) []byte {
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		bytes = append(bytes, byte(mrand.Intn(75)+'0'))
	}

	return bytes
}
