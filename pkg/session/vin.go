package session

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// VIN validation and utilities for machine identification

var (
	// VIN must be exactly 17 characters, alphanumeric (excluding I, O, Q)
	vinRegex = regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`)
	
	// VIN check digit weights
	vinWeights = []int{8, 7, 6, 5, 4, 3, 2, 10, 0, 9, 8, 7, 6, 5, 4, 3, 2}
	
	// VIN character values
	vinValues = map[rune]int{
		'A': 1, 'B': 2, 'C': 3, 'D': 4, 'E': 5, 'F': 6, 'G': 7, 'H': 8,
		'J': 1, 'K': 2, 'L': 3, 'M': 4, 'N': 5, 'P': 7, 'R': 9,
		'S': 2, 'T': 3, 'U': 4, 'V': 5, 'W': 6, 'X': 7, 'Y': 8, 'Z': 9,
		'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
	}
)

// VINInfo contains decoded VIN information
type VINInfo struct {
	VIN              string
	Valid            bool
	WMI              string // World Manufacturer Identifier (positions 1-3)
	VDS              string // Vehicle Descriptor Section (positions 4-9)
	VIS              string // Vehicle Identifier Section (positions 10-17)
	Manufacturer     string // Decoded manufacturer
	ModelYear        int    // Decoded model year
	PlantCode        string // Manufacturing plant
	SequentialNumber string // Sequential production number
	EngineCode       string // Position 8 (engine type)
	CheckDigit       string // Position 9
}

// ValidateVIN checks if a VIN is valid according to ISO 3779 standard
func ValidateVIN(vin string) bool {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	
	// Check format
	if !vinRegex.MatchString(vin) {
		return false
	}
	
	// Verify check digit (position 9)
	return verifyCheckDigit(vin)
}

// DecodeVIN extracts information from a VIN
func DecodeVIN(vin string) (*VINInfo, error) {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	
	if !vinRegex.MatchString(vin) {
		return nil, fmt.Errorf("invalid VIN format: must be 17 alphanumeric characters (excluding I, O, Q)")
	}
	
	info := &VINInfo{
		VIN:              vin,
		Valid:            verifyCheckDigit(vin),
		WMI:              vin[0:3],
		VDS:              vin[3:9],
		VIS:              vin[9:17],
		EngineCode:       string(vin[7]),
		CheckDigit:       string(vin[8]),
		PlantCode:        string(vin[10]),
		SequentialNumber: vin[11:17],
	}
	
	// Decode manufacturer from WMI
	info.Manufacturer = decodeManufacturer(info.WMI)
	
	// Decode model year from position 10
	info.ModelYear = decodeModelYear(vin[9])
	
	return info, nil
}

// verifyCheckDigit validates the VIN check digit (position 9)
func verifyCheckDigit(vin string) bool {
	sum := 0
	for i, char := range vin {
		value, ok := vinValues[char]
		if !ok {
			return false
		}
		sum += value * vinWeights[i]
	}
	
	remainder := sum % 11
	checkChar := vin[8]
	
	if remainder == 10 {
		return checkChar == 'X'
	}
	return checkChar == rune('0'+remainder)
}

// decodeModelYear converts VIN position 10 to model year
func decodeModelYear(char byte) int {
	// Years 1980-2009 use numbers 1-9 then letters A-Y (excluding I, O, Q, U, Z)
	// Years 2010-2039 use A-Y again
	
	yearMap := map[byte]int{
		'A': 1980, 'B': 1981, 'C': 1982, 'D': 1983, 'E': 1984, 'F': 1985, 'G': 1986, 'H': 1987,
		'J': 1988, 'K': 1989, 'L': 1990, 'M': 1991, 'N': 1992, 'P': 1993, 'R': 1994, 'S': 1995,
		'T': 1996, 'V': 1997, 'W': 1998, 'X': 1999, 'Y': 2000,
		'1': 2001, '2': 2002, '3': 2003, '4': 2004, '5': 2005, '6': 2006, '7': 2007, '8': 2008, '9': 2009,
	}
	
	if year, ok := yearMap[char]; ok {
		return year
	}
	
	// For 2010-2039, letters repeat
	year2010Map := map[byte]int{
		'A': 2010, 'B': 2011, 'C': 2012, 'D': 2013, 'E': 2014, 'F': 2015, 'G': 2016, 'H': 2017,
		'J': 2018, 'K': 2019, 'L': 2020, 'M': 2021, 'N': 2022, 'P': 2023, 'R': 2024, 'S': 2025,
		'T': 2026, 'V': 2027, 'W': 2028, 'X': 2029, 'Y': 2030,
	}
	
	if year, ok := year2010Map[char]; ok {
		return year
	}
	
	return 0
}

// decodeManufacturer returns manufacturer name from WMI
func decodeManufacturer(wmi string) string {
	// Common WMI prefixes (partial list)
	manufacturers := map[string]string{
		"1G1": "Chevrolet (USA)",
		"1G4": "Buick (USA)",
		"1GC": "Chevrolet Truck (USA)",
		"1GM": "Pontiac (USA)",
		"1HG": "Honda (USA)",
		"1FA": "Ford (USA)",
		"1FT": "Ford Truck (USA)",
		"1J4": "Jeep (USA)",
		"1N4": "Nissan (USA)",
		"2G1": "Chevrolet (Canada)",
		"2HG": "Honda (Canada)",
		"2T1": "Toyota (Canada)",
		"3FA": "Ford (Mexico)",
		"3VW": "Volkswagen (Mexico)",
		"4T1": "Toyota (USA)",
		"5FN": "Honda (USA)",
		"5YJ": "Tesla (USA)",
		"JHM": "Honda (Japan)",
		"JTD": "Toyota (Japan)",
		"KMH": "Hyundai (Korea)",
		"SAJ": "Jaguar (UK)",
		"SAL": "Land Rover (UK)",
		"SCC": "Lotus (UK)",
		"TRU": "Audi (Hungary)",
		"VF1": "Renault (France)",
		"VF3": "Peugeot (France)",
		"WAU": "Audi (Germany)",
		"WBA": "BMW (Germany)",
		"WDB": "Mercedes-Benz (Germany)",
		"WP0": "Porsche (Germany)",
		"WVW": "Volkswagen (Germany)",
		"YV1": "Volvo (Sweden)",
		"ZFF": "Ferrari (Italy)",
	}
	
	// Try 3-character match first
	if mfr, ok := manufacturers[wmi]; ok {
		return mfr
	}
	
	// Try 2-character match
	if mfr, ok := manufacturers[wmi[:2]]; ok {
		return mfr
	}
	
	// Return country code at minimum
	return decodeCountry(wmi[0])
}

// decodeCountry returns country from first WMI character
func decodeCountry(char byte) string {
	countries := map[byte]string{
		'1': "United States", '2': "Canada", '3': "Mexico",
		'4': "United States", '5': "United States",
		'J': "Japan", 'K': "Korea", 'L': "China",
		'S': "United Kingdom", 'T': "Czech Republic/Hungary",
		'V': "France/Spain", 'W': "Germany", 'Y': "Sweden/Finland",
		'Z': "Italy",
	}
	
	if country, ok := countries[char]; ok {
		return country
	}
	return "Unknown"
}

// GenerateMachineID creates a machine ID for non-VIN equipment
func GenerateMachineID(make, model string, year int, serialNumber string) string {
	// Format: MAKE-MODEL-YEAR-SERIAL
	// Example: HONDA-GX390-2020-12345678
	make = strings.ToUpper(strings.ReplaceAll(make, " ", ""))
	model = strings.ToUpper(strings.ReplaceAll(model, " ", ""))
	serial := strings.ToUpper(strings.ReplaceAll(serialNumber, " ", ""))
	
	return fmt.Sprintf("%s-%s-%d-%s", make, model, year, serial)
}

// ParseMachineID extracts components from a generated machine ID
func ParseMachineID(machineID string) (make, model string, year int, serial string, err error) {
	parts := strings.Split(machineID, "-")
	if len(parts) != 4 {
		return "", "", 0, "", fmt.Errorf("invalid machine ID format")
	}
	
	year, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", "", 0, "", fmt.Errorf("invalid year in machine ID: %w", err)
	}
	
	return parts[0], parts[1], year, parts[3], nil
}

// IsVIN checks if a string looks like a VIN (vs a machine ID)
func IsVIN(id string) bool {
	id = strings.ToUpper(strings.TrimSpace(id))
	return vinRegex.MatchString(id)
}
