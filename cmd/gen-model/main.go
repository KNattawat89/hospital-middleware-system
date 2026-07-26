package main

import (
	"log"
	"os"
	"strings"

	"github.com/bsthun/gut"
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// ==========================ß
// 🔹 Configuration
// ==========================
var (
	Schema        = os.Getenv("GOOSE_SCHEMA")
	ModelPath     = "infra/db/model"
	ExcludeTables = []string{"goose_db_version"}
)

// ==========================
// 🔹 Scan .go Files in Model Path
// - Adds any .go file without a corresponding .gen.go file to ExcludeTables
// ==========================
func scanModelFiles(modelPath string) {
	files, err := os.ReadDir(modelPath)
	if err != nil {
		log.Fatal("❌ Failed to read model directory:", err)
	}

	genFiles := make(map[string]bool)

	// Scan for .gen.go files
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".gen.go") {
			genFiles[strings.TrimSuffix(name, ".gen.go")] = true
		}
	}

	// Add files without .gen.go to ExcludeTables
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, ".gen.go") {
			baseName := strings.TrimSuffix(name, ".go")
			if !genFiles[baseName] {
				ExcludeTables = append(ExcludeTables, baseName)
			}
		}
	}
}

// ==========================
// 🔹 Initialize Database Connection
// ==========================
func connectDB() *gorm.DB {
	dsn := os.Getenv("GOOSE_DBSTRING")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}
	log.Println("✅ Connected to database successfully")
	return db
}

// ==========================
// 🔹 Configure Model Generator
// ==========================
func configureGenerator(db *gorm.DB) *gen.Generator {
	g := gen.NewGenerator(gen.Config{
		OutPath:           ModelPath,
		ModelPkgPath:      "model",
		FieldNullable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface,
	})

	g.UseDB(db)

	// Set table name strategy (only add schema prefix if it's not empty)
	g.WithTableNameStrategy(func(tableName string) string {
		if Schema != "" && Schema != "." {
			return Schema + "." + tableName
		}
		return tableName
	})

	// Set file name strategy
	g.WithFileNameStrategy(func(tableName string) string {
		return tableName
	})

	return g
}

// ==========================
// 🔹 Fetch Tables from Database
// ==========================
func getTables(db *gorm.DB) []string {
	tables, err := db.Migrator().GetTables()
	if err != nil {
		log.Fatal("❌ Failed to get tables:", err)
	}

	// Extract just the table name if schema-qualified
	cleanTables := make([]string, 0, len(tables))
	for _, table := range tables {
		// Remove schema prefix if present
		if strings.Contains(table, ".") {
			parts := strings.Split(table, ".")
			table = parts[len(parts)-1] // Get the last part (table name)
		}
		cleanTables = append(cleanTables, table)
	}

	return cleanTables
}

// ==========================
// 🔹 Generate Models for Tables
// ==========================
func generateModels(g *gen.Generator, tables []string) {
	for _, table := range tables {
		if gut.Contain(ExcludeTables, table) {
			log.Println("🛑 Skipping excluded table:", table)
			continue
		}

		log.Println("🚀 Generating model for table:", table)

		g.GenerateModel(table, gen.FieldModify(func(f gen.Field) gen.Field {
			// * add nil checks to prevent panic
			if f.ColumnName == "" {
				return f
			}

			// * uuid fields
			if (f.Type == "string" || f.Type == "*string") && isUUIDColumn(f.ColumnName) {
				f.Type = "*uuid.UUID"
				return f
			}

			// * everything else: convert to pointer
			switch f.Type {
			case "string":
				f.Type = "*string"
			case "int", "int32", "int64":
				f.Type = "*" + f.Type
			case "float32", "float64":
				f.Type = "*" + f.Type
			case "time.Time":
				f.Type = "*time.Time"
			}

			return f
		}))

		log.Println("✅ Model generated for table:", table)
	}
}

func isUUIDColumn(column string) bool {
	column = strings.ToLower(column)
	return strings.HasSuffix(column, "id") || column == "id"
}

// ==========================
// 🔹 Main Execution
// ==========================
func main() {
	log.Println("🔄 Scanning model files...")
	scanModelFiles(ModelPath)

	log.Println("🔄 Connecting to database...")
	db := connectDB()

	log.Println("🔄 Configuring generator...")
	g := configureGenerator(db)

	log.Println("🔄 Fetching tables...")
	tables := getTables(db)

	log.Println("🔄 Generating models...")
	generateModels(g, tables)

	log.Println("🚀 Executing generator...")
	g.Execute()

	log.Println("🎉 Model generation complete!")
}
