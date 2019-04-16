package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
	"strings"

	"github.com/schollz/progressbar"
)

type RangeProb struct {
	lo   int64
	hi   int64
	prob float64
}

type Input struct {
	MemCount int64
	Threshold int64
	SimCount int64
	RandProbs [][]float64
	RandAmounts [][]int64
}

type SimRun struct {
	SimNum int64
	MemNum int64
	Amount int64
}

type Results struct {
	Hits []SimRun
	Sims []SimRun
	mux  sync.Mutex
}

func (i *Input) Get() {
	i.MemCount = requestIntInput("Enter number of members: ")
	i.Threshold = requestIntInput("Enter threshold: ")
	i.SimCount = 1000
}

func (i *Input) GenerateSeeds() {
	var rows  []string
	bar := progressbar.New(int(i.SimCount * i.MemCount))
	for s := int64(0); s < i.SimCount; s++ {
		var probs []float64
		var amounts []int64
		for m := int64(0); m < i.MemCount; m++ {
			bar.Add(1)
			randSource := rand.NewSource(time.Now().UnixNano() + s + m) 
			prob := rand.New(randSource).Float64()
			probs = append(probs, prob)
			amount := rand.New(randSource).Int63n(2500000)
			amounts = append(amounts, amount)
			rows = append(rows, fmt.Sprintf("%v,%v", prob, amount))
		}
		i.RandProbs = append(i.RandProbs, probs)
		i.RandAmounts = append(i.RandAmounts, amounts)
	}
	bar.Finish()
	fmt.Println("")
}

func (s *SimRun) Less(other SimRun) bool {
	return s.SimNum < other.SimNum || (s.SimNum == other.SimNum && s.MemNum < other.MemNum)
}

func (r *Results) Prep() {
	sort.Slice(r.Hits, func(i, j int) bool {
		return r.Hits[i].Less(r.Hits[j])
	})

	sort.Slice(r.Sims, func(i, j int) bool {
		return r.Sims[i].Less(r.Sims[j])
	})
}

func (r *Results) Save(input *Input, path string) {
	_ = os.Mkdir("results", os.ModePerm)
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	metaInfo := fmt.Sprintf("Members:,%d,,Threshold:,%d,,Simulations:,%d\n\n", input.MemCount, input.Threshold, input.SimCount)
	header := "Simulation Number,Result,,,Simulation Number,Claim ID,Claim Amount\n"
	var rows []string
	topLength := MaxInt(len(r.Sims), len(r.Hits))
	for i := 0; i < topLength; i++ {
		var row string
		if i < len(r.Sims) {
			sr := r.Sims[i]
			row = fmt.Sprintf("%d,%d", sr.SimNum, sr.Amount)
		} else {
			row = ","
		}
		if i < len(r.Hits) {
			hit := r.Hits[i]
			row += fmt.Sprintf(",,,%d,%d,%d", hit.SimNum, hit.MemNum, hit.Amount)
		}
		rows = append(rows, row)
	}
	data := strings.Join(rows, "\n")
	_, err = f.WriteString(metaInfo+header+data)
	if err != nil {
		panic(err)
	}
}

func MaxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func readProbs() ([]RangeProb, error) {
	csvFile, _ := os.Open("range_probabilities.csv")
	reader := csv.NewReader(bufio.NewReader(csvFile))
	var probs []RangeProb
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return []RangeProb{}, err
		}
		lo, _ := strconv.ParseInt(line[0], 0, 64)
		hi, _ := strconv.ParseInt(line[1], 0, 64)
		prob, _ := strconv.ParseFloat(line[2], 64)
		probs = append(probs, RangeProb{
			lo:   lo,
			hi:   hi,
			prob: prob,
		})
	}
	return probs, nil
}

func prep(rangeProbs []RangeProb) ([]RangeProb, error) {
	var cumRangeProbs []RangeProb
	sort.Slice(rangeProbs, func(i, j int) bool {
		return rangeProbs[i].prob > rangeProbs[j].prob
	})
	var probSum float64
	for _, rp := range rangeProbs {
		probSum += rp.prob
		cumRangeProbs = append(cumRangeProbs, RangeProb{rp.lo, rp.hi, probSum})
	}
	epsilon := 0.000000000001
	if math.Abs(1.0 - probSum) > epsilon {
		return cumRangeProbs, fmt.Errorf("all probabilities should add up very close to 1.0 but got sum of %v instead", probSum)
	}
	return cumRangeProbs, nil
}

func requestIntInput(prompt string) int64 {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	num, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		fmt.Printf("Expected integer but got %s, try again...\n", text)
		return requestIntInput(prompt)
	}
	return num
}

func pickRange(cumRangeProbs []RangeProb, randProp float64) (RangeProb, error) {
	for _, crp := range cumRangeProbs {
		if randProp < crp.prob {
			return crp, nil
		}
	}
	return RangeProb{}, fmt.Errorf("no range found")
}

func pickAmount(cumRangeProbs []RangeProb, randProb float64, randAmount int64) (int64, error) {
	rp, err := pickRange(cumRangeProbs, randProb)
	if err != nil {
		fmt.Printf("%v\n", err.Error())
		return 0, err
	}
	if rp.lo == rp.hi {
		return rp.lo, nil
	}
	amount := rp.lo + (randAmount % (1 + rp.hi - rp.lo))
	return amount, nil
}

func runCompany(simNum int64, input *Input, cumRangeProbs []RangeProb, results *Results, wg *sync.WaitGroup) {
	defer wg.Done()
	var amount int64
	var sumAmount int64
	for j := int64(0); j < input.MemCount; j++ {
		randProb := input.RandProbs[simNum][j]
		randAmount := input.RandAmounts[simNum][j]
		amount, _ = pickAmount(cumRangeProbs, randProb, randAmount)
		sumAmount += amount
		if amount >= input.Threshold {
			hit := SimRun{SimNum: simNum+1, MemNum: j+1, Amount: amount}
			results.mux.Lock()
			results.Hits = append(results.Hits, hit)
			results.mux.Unlock()
		}
	}
	simRun := SimRun{SimNum: simNum+1, Amount: sumAmount}
	results.mux.Lock()
	results.Sims = append(results.Sims, simRun)
	results.mux.Unlock()
}

func runSimulation(input *Input, cumRangeProbs []RangeProb, results *Results) {
	var wg sync.WaitGroup
	wg.Add(int(input.SimCount))
	for i := int64(0); i < input.SimCount; i++ {
		go runCompany(i, input, cumRangeProbs, results, &wg)
	}
	wg.Wait()
}

func main() {
	var input Input
	input.Get()
	fmt.Println("Running simulation...")
	input.GenerateSeeds()
	rangeProbs, _ := readProbs()
	cumRangeProbs, err := prep(rangeProbs)
	if err != nil {
		panic(err)
	}
	var results Results
	runSimulation(&input, cumRangeProbs, &results)
	resultsPath := fmt.Sprintf("results/simulation_results_%d.csv", time.Now().Unix())
	results.Prep()
	results.Save(&input, resultsPath)
	fmt.Printf("Results are saved in %s\n", resultsPath)
}
