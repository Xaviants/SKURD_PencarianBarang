package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Item struct untuk menyimpan data barang
type Item struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

// Daftar barang (menggunakan slice sebagai array dinamis)
var items = []Item{
	{ID: 1, Name: "Laptop", Price: 12000000},
	{ID: 2, Name: "Smartphone", Price: 8000000},
	{ID: 3, Name: "Headphones", Price: 200000},
	{ID: 4, Name: "PS 4", Price: 2500000},
}

// Log aktivitas (menggunakan linked list)
var activityLog = list.New()

// Pencarian cepat menggunakan map
var itemIndex = map[string]int{}

// Inisialisasi indeks item
func init() {
	for _, item := range items {
		itemIndex[item.Name] = item.ID
	}
}

// Fungsi pencarian barang (GET)
func searchItems(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("query"))
	var results []Item
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), query) {
			results = append(results, item)
		}
	}
	activityLog.PushBack(fmt.Sprintf("Search query: %s", query))
	jsonResponse(w, results)
}

// Fungsi menambahkan barang baru (POST)
func addItem(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	newItem.ID = len(items) + 1
	items = append(items, newItem)
	itemIndex[newItem.Name] = newItem.ID
	activityLog.PushBack(fmt.Sprintf("Added item: %s", newItem.Name))
	jsonResponse(w, newItem)
}

// Fungsi menghapus barang berdasarkan ID (DELETE)
func deleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	var found bool
	for i, item := range items {
		if item.ID == id {
			items = append(items[:i], items[i+1:]...)
			delete(itemIndex, item.Name)
			found = true
			activityLog.PushBack(fmt.Sprintf("Deleted item ID: %d", id))
			break
		}
	}
	if !found {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Fungsi melihat log aktivitas (GET)
func getActivityLog(w http.ResponseWriter, r *http.Request) {
	var logs []string
	for e := activityLog.Front(); e != nil; e = e.Next() {
		logs = append(logs, e.Value.(string))
	}
	jsonResponse(w, logs)
}

// Fungsi respons JSON
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Fungsi utama menjalankan server
func main() {
	http.HandleFunc("/items/search", searchItems)
	http.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			addItem(w, r)
		case http.MethodDelete:
			deleteItem(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/activity-log", getActivityLog)

	fmt.Println("Server berjalan di http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}