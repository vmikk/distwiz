# Sparse to full matrix converter

## Overview

This utility converts a sparse matrix 
(such as those produced by `usearch -calc_distmx`) 
into a full square matrix.

## Notes and limitations

This project is experimental and may require further optimization to enhance performance.
Selection between the in-memory or disk-based processing methods 
is based on the number of unique labels in the input data (10,000 sequences by default).  

- The in-memory solution requires approximately 100GB of RAM for 30,000 objects (equivalent to around 450 million pairwise distances)  
- The disk-based solution, while significantly more memory-efficient, is I/O intensive and much slower.  

## Usage

```shell
distwiz --input mx.txt --output dist.txt.gz
```

Supported arguments:  
- `--input`: Path to the input file containing the sparse distance matrix  
- `--output`: Path to the output file (GZIP-compressed)  
- `--mode`: Processing mode: `auto`, `mem` (in-memory), or `disk` (disk-based). Default is `auto`  
- `--compresslevel`: GZIP compression level (1-9). The default is 4  

## Installation

Compile the program using `go build`.
