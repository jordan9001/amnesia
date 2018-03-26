package amnesia

import (
	"encodeing/base64"
	"math/rand"
)

type HEntry struct {
	prob Int
	f    func([]byte) []byte
}

// table that balances the amount we do each operation
var hTable []HEntry = []HEntry{
	{1, bad_add},
	{1, bad_replace},
	{5, rearrange_random},
	{9, remove_random},
	{15, add_random},
	{9, replace_random},
	{3, repeat_add},
	{4, repeat_replace},
}
var hTableTotal int

func init() {
	// initialize hTableTotal
	setHTableTotal()
	// initialize BLNS from BLNS_b64

	BLNS = make([][]byte, len(BLNS_b64))
	for i, v := range BLNS_b64 {
		bad, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			log.Fatalf("Error while decoding BLNS_b64 entry %d\n", i)
		}
		BLSN[i] = bad
	}
}

func setHTableTotal() {
	hTableTotal = 0
	for _, v := range hTable {
		hTableTotal += v.prob
	}
}

// can also be used to add custom mutations
func SetHeruistics(hvals []HEntry) {
	hTable = hvals
	// reinit table total
	setHTableTotal()
}

// For mutating known input to reveal bad stuff
func Mutate(src string, level int) string {

	s := []byte(src)

	for ; level > 0; level-- {
		p := rand.Intn(hTableTotal)
		t := 0
		for _, v := range hTable {
			t += v.prob
			if p < t {
				s = v.f(s)
				break
			}
		}
	}

	return string(s)
}

// change operations TODO
// - find ascii numbers and mess them up

func bad_add(in []byte) []byte {
	bad := BLNS[rand.Intn(len(BLNS))]
	sz := len(bad)
	pos := rand.Intn(len(in) + 1)

	// insert
	out := append(in, make([]byte, sz))
	copy(out[pos+sz:], in[pos:])
	copy(out[pos:], bad)
}

func bad_replace(in []byte) []byte {
	bad := BLNS[rand.Intn(len(BLNS))]
	if len(in) < len(bad) {
		// silent fail
		return in
	}

	pos := rand.Intn(len(in) - len(bad) + 1)

	copy(in[pos:], bad)

	return in
}

func rearrange_random(in []byte) []byte {
	pos := rand.Intn(len(in))
	sz := rand.Intn(len(in)-pos-1) + 1

	dst := rand.Intn(len(in) + 1 - sz)
	tblock := make([]byte, sz)
	copy(tblock, in[pos:])

	copy(in[dst:dst+sz], in[pos:])
	copy(in[pos:], tblock)

	return in
}

func remove_random(in []byte) []byte {
	pos := rand.Intn(len(in)-1) + 1
	out := append(in[:pos], in[pos-1:]...)
	return out
}

func add_random(in []byte) []byte {
	pos := rand.Intn(len(in) + 1)
	val := byte(rand.Int())

	out := append(in, 0)
	copy(out[pos+1:], in[pos:])
	out[pos] = val
	return out
}

func replace_random(in []byte) []byte {
	pos := rand.Intn(len(in))
	val := byte(rand.Int())
	in[pos] = val
	return in
}

func repeat_add(in []byte) []byte {
	// take a random selection
	start := rand.Intn(len(in))
	end := rand.Intn(len(in)-start) + start + 1
	size := end - start

	// take a random offset
	pos := rand.Intn(len(in) + 1)

	// insert
	out := append(in, make([]byte, size))
	copy(out[pos+size:], in[pos:])
	copy(out[pos:], in[start:end])
	return out
}

func repeat_replace(in []byte) []byte {
	// take a random selection
	pos := rand.Intn(len(in))
	sz := rand.Intn(len(in)-pos-1) + 1

	dst := rand.Intn(len(in) + 1 - sz)

	copy(in[dst:dst+sz], in[pos:])

	return in
}
