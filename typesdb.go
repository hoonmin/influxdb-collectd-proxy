package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Types map[string][]string

// Parses types.db file.
// The result map looks like this:
// 	"ps_count" -> [["processes" "GAUGE" "0" "1000000"] ["threads" "GAUGE" "0" "1000000"]]
func ParseTypesDB(path string) (Types, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(Types)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(strings.Replace(scanner.Text(), "\t", " ", -1), " ")
		if len(fields) < 2 {
			continue
		}

		typeName := fields[0]
		if string(typeName[0]) == "#" {
			continue
		}

		types := []string{}
		for _, v := range fields[1:] {
			if len(v) == 0 {
				continue
			}

			v = strings.Trim(v, ",")

			vFields := strings.Split(v, ":")
			if len(vFields) != 4 {
				log.Printf("cannot parse data source %x on type %x\n", v, typeName)
				continue
			}

			types = append(types, vFields...)
		}
		result[typeName] = types
	}

	return result, nil
}
