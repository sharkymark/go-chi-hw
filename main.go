package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

type Commission struct {
	ID            int64
	VariableRate  float64
	CommissionAmt float64
	Attainment    float64
	VariableComp  float64
	Quota         float64
	DealRevenue   float64
	CreatedAt     time.Time
}

var db *sql.DB

func main() {

	// Database connection
	var err error
	db, err = sql.Open("sqlite3", "./db/commissions.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create the commissions table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS commissions (id INTEGER PRIMARY KEY, variable_rate REAL, commission_amt REAL, attainment REAL, variable_comp REAL, quota REAL, deal_revenue REAL, created_at TIMESTAMP)")
	if err != nil {
		panic(err)
	}

	// Chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Routes
	r.Get("/", HomeHandler)
	r.Post("/", SubmitCommissionHandler)
	r.Post("/delete-row", DeleteCommissionHandler)

	// Serve static files
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	FileServer(r, "/static", filesDir)

	// Start the server
	fmt.Println("Starting commissions server on :8080")
	http.ListenAndServe(":8080", r)

}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit URL parameters.")
	}

	// Strip '/static/' from the path
	fs := http.StripPrefix(path, http.FileServer(root))

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	// Note that we are using chi's router here to register the route
	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieve commissions from the database
	rows, err := db.Query("SELECT * FROM commissions")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var commissions []Commission
	for rows.Next() {
		var c Commission
		if err := rows.Scan(&c.ID, &c.VariableRate, &c.CommissionAmt, &c.Attainment, &c.VariableComp, &c.Quota, &c.DealRevenue, &c.CreatedAt); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		commissions = append(commissions, c)
	}

	tmpl := template.New("index.html").Funcs(template.FuncMap{
		"formatAsPercent": func(v float64) string {
			return fmt.Sprintf("%.2f%%", v*100)
		},
		"formatMoney": func(amount float64) string {
			return fmt.Sprintf("$%.2f", amount)
		},
	})

	// Correctly initialize and parse the template
	tmpl, err = tmpl.ParseFiles("./templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Execute the template with the commissions data
	tmpl.Execute(w, commissions)
}

func SubmitCommissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse form values
		r.ParseForm()
		quota, _ := strconv.ParseFloat(r.Form["quota"][0], 64)
		variableComp, _ := strconv.ParseFloat(r.Form["variable_comp"][0], 64)
		dealRevenue, _ := strconv.ParseFloat(r.Form["deal_revenue"][0], 64)

		// Calculate the variable rate
		variableRate := variableComp / quota

		// Calculate the commission amount
		commissionAmt := variableRate * dealRevenue

		// Calculate attainment
		attainment := (dealRevenue / quota) * 100

		// Save to database
		log.Println("Attempting to insert commission into database")
		_, err := db.Exec("INSERT INTO commissions (variable_rate, commission_amt, attainment, variable_comp, quota, deal_revenue, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			variableRate, commissionAmt, attainment, variableComp, quota, dealRevenue, time.Now())
		if err != nil {
			log.Println("Error inserting commission into database:", err)
			http.Error(w, err.Error(), 500)
			return
		}
		log.Println("Successfully inserted commission into database")
		// Redirect to home page to display updated list
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func DeleteCommissionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse form values
		r.ParseForm()
		commissionIDs := r.Form["commission_ids"]
		for _, idStr := range commissionIDs {
			id, _ := strconv.ParseInt(idStr, 10, 64)
			// Delete commission
			log.Println("Attempting to delete commission from database")
			_, err := db.Exec("DELETE FROM commissions WHERE id = ?", id)
			if err != nil {
				log.Println("Error deleting commission from database:", err)
				http.Error(w, err.Error(), 500)
				return
			}
			log.Println("Successfully deleted commission from database")
		}

		// Redirect back to the index page
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
