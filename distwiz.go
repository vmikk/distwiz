package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

func main() {
	// Parse command-line arguments
	inputPath := flag.String("input", "", "Path to the input file containing the sparse distance matrix.")
	outputPath := flag.String("output", "", "Path to the output gzip-compressed file.")
	compresslevel := flag.Int("compresslevel", 4, "GZIP compression level (1-9). Default is 4.")
	flag.Parse()

	// Validate arguments
	if *inputPath == "" || *outputPath == "" {
		log.Fatal("Both input and output paths are required.")
	}

	// Process the sparse matrix
	distances, labels := readSparseMatrix(*inputPath)
	writeSquareMatrix(*outputPath, distances, labels, *compresslevel)
}

func readSparseMatrix(inputPath string) (map[[2]string]float64, []string) {
	distances := make(map[[2]string]float64)
	labelsSet := make(map[string]struct{})

	file, err := os.Open(inputPath)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			log.Fatal("Invalid input format. Each line must contain two labels and a distance.")
		}
		label1, label2, distance := parts[0], parts[1], parts[2]
		dist, err := strconv.ParseFloat(distance, 64)
		if err != nil {
			log.Fatalf("Failed to parse distance '%s' as float: %v", distance, err)
		}
		distances[[2]string{label1, label2}] = dist
		labelsSet[label1] = struct{}{}
		labelsSet[label2] = struct{}{}
	}

	// Convert labels set to sorted slice
	var labels []string
	for label := range labelsSet {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	return distances, labels
}

func writeSquareMatrix(outputPath string, distances map[[2]string]float64, labels []string, compresslevel int) {
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	gz, err := gzip.NewWriterLevel(file, compresslevel)
	if err != nil {
		log.Fatalf("Failed to create gzip writer: %v", err)
	}
	defer gz.Close()

	writer := bufio.NewWriter(gz)
	defer writer.Flush()

	// Write header
	_, err = writer.WriteString(strings.Join(labels, "\t") + "\n")
	if err != nil {
		log.Fatalf("Failed to write to output file: %v", err)
	}

	// Write rows
	for _, label1 := range labels {
		var row []string
		for _, label2 := range labels {
			var distance float64 = 1.0 // Default distance
			if label1 == label2 {
				distance = 0.0
			} else if dist, found := distances[[2]string{label1, label2}]; found {
				distance = dist
			}
			row = append(row, fmt.Sprintf("%.1f", distance))
		}
		_, err := writer.WriteString(strings.Join(row, "\t") + "\n")
		if err != nil {
			log.Fatalf("Failed to write row to output file: %v", err)
		}
	}
}
