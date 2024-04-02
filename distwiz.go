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

// Threshold for deciding between processing methods (in-memory or disk-based)
const LargeNThreshold = 10000

func main() {
	// Parse command-line arguments
	inputPath := flag.String("input", "", "Path to the input file containing the sparse distance matrix.")
	outputPath := flag.String("output", "", "Path to the output gzip-compressed file.")
	compressLevel := flag.Int("compresslevel", 4, "GZIP compression level (1-9). Default is 4.")
	flag.Parse()

	// Validate arguments
	if *inputPath == "" || *outputPath == "" {
		log.Fatal("Both input and output paths are required.")
	}

	labels, err := scanForLabels(*inputPath)
	if err != nil {
		log.Fatalf("Error scanning for labels: %v", err)
	}

	// Choose processing method based on the number of labels
	if len(labels) > LargeNThreshold {
		if err := writeSquareMatrix(*outputPath, *inputPath, labels, *compressLevel); err != nil {
			log.Fatalf("Error writing square matrix: %v", err)
		}
	} else {
		if err := writeSquareMatrixInMemory(*outputPath, *inputPath, labels, *compressLevel); err != nil {
			log.Fatalf("Error writing square matrix using in-memory method: %v", err)
		}
	}
}

// Scan the input file for all unique labels
func scanForLabels(inputPath string) ([]string, error) {
	labelsSet := make(map[string]struct{})
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue // Skip invalid lines
		}
		labelsSet[parts[0]] = struct{}{}
		labelsSet[parts[1]] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	labels := make([]string, 0, len(labelsSet))
	for label := range labelsSet {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels, nil
}

// Write the square matrix by processing each label
func writeSquareMatrix(outputPath, inputPath string, labels []string, compressLevel int) error {
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

	// Write header
	if _, err := writer.WriteString(strings.Join(labels, "\t") + "\n"); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	for _, label1 := range labels {
		if err := writeRow(writer, inputPath, label1, labels); err != nil {
			return err // Error already wrapped
		}
	}

	return nil
}

// Write a single row of the square matrix for a specific label
// (without loading the entire file in memory)
func writeRow(writer *bufio.Writer, inputPath, label1 string, labels []string) error {
	distances := make(map[string]float64)
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file for reading distances: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			continue // Skip invalid lines
		}
		if parts[0] == label1 || parts[1] == label1 {
			dist, err := strconv.ParseFloat(parts[2], 64)
			if err != nil {
				continue // Skip lines with invalid distances
			}
			if parts[0] == label1 {
				distances[parts[1]] = dist
			} else {
				distances[parts[0]] = dist
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading distances for label %s: %w", label1, err)
	}

	var row []string
	for _, label2 := range labels {
		if label1 == label2 {
			row = append(row, "0.0")
		} else if dist, found := distances[label2]; found {
			row = append(row, fmt.Sprintf("%.1f", dist))
		} else {
			row = append(row, "1.0") // Default distance
		}
	}
	if _, err := writer.WriteString(strings.Join(row, "\t") + "\n"); err != nil {
		return fmt.Errorf("failed to write row for label %s: %w", label1, err)
	}

	return nil
}

// Write the square matrix using in-memory data for small number of labels
func writeSquareMatrixInMemory(outputPath, inputPath string, labels []string, compressLevel int) error {
	// Open and read the entire input file into memory
	data, err := readInputFileIntoMemory(inputPath)
	if err != nil {
		return err
	}

	// Proceed with file and gzip creation
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

	// Write header
	if _, err := writer.WriteString(strings.Join(labels, "\t") + "\n"); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	// Process each label using the in-memory data
	for _, label1 := range labels {
		if err := writeRowInMemory(writer, label1, labels, data); err != nil {
			return err // Error already wrapped
		}
	}

	return nil
}

// Read the input file and stores distances in a nested map
func readInputFileIntoMemory(inputPath string) (map[string]map[string]float64, error) {
	data := make(map[string]map[string]float64)
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			continue // Skip invalid lines
		}
		label1, label2 := parts[0], parts[1]
		distance, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			continue // Skip lines with invalid distances
		}

		if data[label1] == nil {
			data[label1] = make(map[string]float64)
		}
		if data[label2] == nil {
			data[label2] = make(map[string]float64)
		}

		// Distance matrix is symmetric
		data[label1][label2] = distance
		data[label2][label1] = distance
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return data, nil
}

// Write a single row for label1 using the in-memory data structure
func writeRowInMemory(writer *bufio.Writer, label1 string, labels []string, data map[string]map[string]float64) error {
	var row []string
	for _, label2 := range labels {
		if label1 == label2 {
			row = append(row, "0.0")
			continue
		}

		if dist, found := data[label1][label2]; found {
			row = append(row, fmt.Sprintf("%.1f", dist))
		} else {
			row = append(row, "1.0") // Default distance if not found
		}
	}

	if _, err := writer.WriteString(strings.Join(row, "\t") + "\n"); err != nil {
		return fmt.Errorf("failed to write row for label %s: %w", label1, err)
	}

	return nil
}
