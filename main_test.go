package main

import (
	"testing"
	"math"
	"math/rand"
	"time"
	"fmt"
)

func getTestProbs() []RangeProb {
	return []RangeProb{
		RangeProb{lo: 1, hi: 4, prob: 0.5},
		RangeProb{lo: 5, hi: 6, prob: 0.25},
		RangeProb{lo: 7, hi: 8, prob: 0.25},
	}
}

func TestPrep(t *testing.T) {
	testProbs := getTestProbs()
	cumProbs, err := prep(testProbs)
	if err != nil {
		t.Error(err)
	}
	
	expectedCumProbs := []float64{0.5,0.75,1.0}
	for i := 0; i < 3; i++ {
		if !floatEq(cumProbs[i].prob, expectedCumProbs[i]) {
			t.Errorf("expected cum. prob of %v, but got %v", expectedCumProbs[i], cumProbs[i].prob)
		}
	}
}


func TestPickAmount(t *testing.T) {
	testProbs := getTestProbs()
	cumProbs, err := prep(testProbs)
	if err != nil {
		t.Error(err)
	}
	freqs := make(map[int64]int)
	for i := int64(1); i < 9; i++ {
		freqs[i] = 0
	}
	for i := int64(0); i < 100000; i++ {
		randSource := rand.NewSource(time.Now().UnixNano() + i) 
		randProb := rand.New(randSource).Float64()
		amount, err := pickAmount(cumProbs, randProb)
		if err != nil {
			t.Error(err)
		}
		freqs[amount]++
	}
	mx := 0
	mn := int(math.MaxInt64)
	for _, f := range freqs {
		if f > mx {
			mx = f
		}
		if f < mn {
			mn = f
		}
	}
	fmt.Println(freqs)
	percentDiff := 100.0 * float64(mx - mn) / float64(mx)
	if percentDiff > 5 {
		t.Errorf("expected freq to be same but min and max freqs differ significantly by %v percent", percentDiff)
	}
}


func floatEq(f1 float64, f2 float64) bool {
	epsilon := 0.0000000000001
	return math.Abs(f1 - f2) < epsilon
}

