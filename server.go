package main

import (
	"container/list"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
	{ID: 5, Name: "PS 5", Price: 5500000},
}

// Log aktivitas (menggunakan linked list)
var activityLog = list.New()

// Barang terbaru (menggunakan queue dengan linked list)
var recentItemsQueue = list.New()

// Maksimum barang dalam queue
const maxRecentItems = 5

// Fungsi menambahkan item ke queue
func enqueueRecentItem(item Item) {
	if recentItemsQueue.Len() >= maxRecentItems {
		recentItemsQueue.Remove(recentItemsQueue.Front()) // Hapus elemen terdepan jika penuh
	}
	recentItemsQueue.PushBack(item)
}

// Pencarian cepat menggunakan map
var itemIndex = map[string]int{}

// Inisialisasi indeks item
func init() {
	for _, item := range items {
		itemIndex[item.Name] = item.ID
	}
}

// Fungsi pencarian barang (GET)
func searchItems(c *gin.Context) {
	query := strings.ToLower(c.Query("query"))
	var results []Item
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), query) {
			results = append(results, item)
		}
	}
	activityLog.PushBack(fmt.Sprintf("Searched item: %s", query))
	c.JSON(http.StatusOK, results)
}

// Fungsi menambahkan barang baru (POST)
func addItem(c *gin.Context) {
	var newItem Item
	if err := c.ShouldBindJSON(&newItem); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	newItem.ID = len(items) + 1
	items = append(items, newItem)
	itemIndex[newItem.Name] = newItem.ID
	enqueueRecentItem(newItem) // Tambahkan ke queue barang terbaru
	activityLog.PushBack(fmt.Sprintf("Added item: %s", newItem.Name))
	c.JSON(http.StatusCreated, newItem)
}

// Fungsi menghapus barang berdasarkan ID (DELETE)
func deleteItem(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Fungsi melihat barang terbaru (GET)
func getRecentItems(c *gin.Context) {
	var recentItems []Item
	for e := recentItemsQueue.Front(); e != nil; e = e.Next() {
		recentItems = append(recentItems, e.Value.(Item))
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   recentItems,
	})
}

// Fungsi melihat log aktivitas (GET)
func getActivityLog(c *gin.Context) {
	var logs []string
	for e := activityLog.Front(); e != nil; e = e.Next() {
		logs = append(logs, e.Value.(string))
	}
	c.JSON(http.StatusOK, logs)
}

func main() {
	router := gin.Default()

	// Rute untuk pencarian barang
	router.GET("/items/search", searchItems)

	// Rute untuk menambahkan barang baru
	router.POST("/items", addItem)

	// Rute untuk menghapus barang
	router.DELETE("/items", deleteItem)

	// Rute untuk melihat barang terbaru
	router.GET("/items/recent", getRecentItems)

	// Rute untuk melihat log aktivitas
	router.GET("/activity-log", getActivityLog)

	fmt.Println("Server berjalan di http://localhost:8080")
	router.Run(":8080")
}
