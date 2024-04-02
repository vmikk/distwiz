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
	inputPath, outputPath, compressLevel := parseFlags()

	distances, labels, err := readSparseMatrix(inputPath)
	if err != nil {
		log.Fatalf("Error reading sparse matrix: %v", err)
	}

	if err := writeSquareMatrix(outputPath, distances, labels, compressLevel); err != nil {
		log.Fatalf("Error writing square matrix: %v", err)
	}
}

func parseFlags() (inputPath, outputPath string, compressLevel int) {
	flag.StringVar(&inputPath, "input", "", "Path to the input file containing the sparse distance matrix.")
	flag.StringVar(&outputPath, "output", "", "Path to the output gzip-compressed file.")
	flag.IntVar(&compressLevel, "compresslevel", 4, "GZIP compression level (1-9). Default is 4.")
	flag.Parse()

	if inputPath == "" || outputPath == "" {
		log.Fatal("Both input and output paths are required.")
	}
	return
}

func readSparseMatrix(inputPath string) (map[[2]string]float64, []string, error) {
	distances := make(map[[2]string]float64)
	labelsSet := make(map[string]struct{})

	file, err := os.Open(inputPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			return nil, nil, fmt.Errorf("invalid input format. Each line must contain two labels and a distance")
		}
		label1, label2, distance := parts[0], parts[1], parts[2]
		dist, err := strconv.ParseFloat(distance, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse distance '%s' as float: %w", distance, err)
		}
		distances[[2]string{label1, label2}] = dist
		labelsSet[label1] = struct{}{}
		labelsSet[label2] = struct{}{}
	}

	var labels []string
	for label := range labelsSet {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	return distances, labels, nil
}

func writeSquareMatrix(outputPath string, distances map[[2]string]float64, labels []string, compressLevel int) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	gz, err := gzip.NewWriterLevel(file, compressLevel)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gz.Close()

	writer := bufio.NewWriter(gz)
	defer writer.Flush()

	if _, err := writer.WriteString(strings.Join(labels, "\t") + "\n"); err != nil {
		return fmt.Errorf("failed to write header to output file: %w", err)
	}

	for _, label1 := range labels {
		var row []string
		for _, label2 := range labels {
			distance := 1.0 // Default distance
			if label1 == label2 {
				distance = 0.0
			} else if dist, found := distances[[2]string{label1, label2}]; found {
				distance = dist
			}
			row = append(row, fmt.Sprintf("%.1f", distance))
		}
		if _, err := writer.WriteString(strings.Join(row, "\t") + "\n"); err != nil {
			return fmt.Errorf("failed to write row to output file: %w", err)
		}
	}
	return nil
}
