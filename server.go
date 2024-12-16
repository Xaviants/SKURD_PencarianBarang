package main

import (
	"container/list"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Struct untuk barang
type Item struct {
	ID    int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name  string `gorm:"size:255;not null" json:"name"`
	Price int    `gorm:"not null" json:"price"`
}

// Koneksi database
var db *gorm.DB

// Log aktivitas (menggunakan linked list)
var activityLog = list.New()

// Barang terbaru (menggunakan queue dengan linked list)
var recentItemsQueue = list.New()

const maxRecentItems = 5

// Fungsi menambahkan item ke queue
func enqueueRecentItem(item Item) {
	if recentItemsQueue.Len() >= maxRecentItems {
		recentItemsQueue.Remove(recentItemsQueue.Front())
	}
	recentItemsQueue.PushBack(item)
}

// Fungsi inisialisasi database
func initDB() {
	// Konfigurasi koneksi ke MySQL (XAMPP)
	dsn := "root:@tcp(127.0.0.1:3306)/db_barang" // Format: user:password@tcp(localhost:port)/database
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// Membuat tabel jika belum ada
	err = db.AutoMigrate(&Item{})
	if err != nil {
		log.Fatalf("Error migrating database: %v", err)
	}
}

// Fungsi pencarian barang berdasarkan nama
func searchItems(c *gin.Context) {
	query := strings.TrimSpace(strings.ToLower(c.Query("query")))
	if strings.ContainsAny(query, "!@#$%^&*()<>/?;:'\"[]{}\\|+=-_`~,.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid characters in query"})
		return
	}

	var results []Item
	db.Where("LOWER(name) LIKE ?", "%"+query+"%").Find(&results)

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"query": query,
			"status": "error",
			"message": "Item cannot be found",
		})
		return
	}

	// Log search item(s) activity
	activityLog.PushBack(fmt.Sprintf("Searched item: %s", query))
	c.JSON(http.StatusOK, gin.H{
		"query":  query,
		"status": "success",
		"data":   results,
	})
}

// Function filter searched items by alphabet
func searchItemsAlphabetically(c *gin.Context) {
	query := strings.TrimSpace(strings.ToLower(c.Query("query")))
	if strings.ContainsAny(query, "!@#$%^&*()<>/?;:'\"[]{}\\|+=-_`~,.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid characters in query"})
		return
	}

	var results []Item
	db.Where("LOWER(name) LIKE ?", "%"+query+"%").Order("name ASC").Find(&results)

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"query": query,
			"status": "error",
			"message": "Item cannot be found",
		})
		return
	}

	// Log search item(s) activity
	activityLog.PushBack(fmt.Sprintf("Searched item by alphabet: %s", query))
	c.JSON(http.StatusOK, gin.H{
		"query":  query,
		"status": "success",
		"message": "Searched items filtered by alphabet",
		"data":   results,
	})
}

// Fungsi pencarian barang berdasarkan rentang harga
func searchItemsByPriceRange(c *gin.Context) {
	minPrice, errMin := strconv.Atoi(c.DefaultQuery("minPrice", "0"))
	maxPrice, errMax := strconv.Atoi(c.DefaultQuery("maxPrice", "0"))

	if errMin != nil || errMax != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid price range"})
		return
	}

	var results []Item
	db.Where("price >= ? AND (price <= ? OR ? = 0)", minPrice, maxPrice, maxPrice).Find(&results)

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "error",
			"message": "There's no available item in that range of prices.",
		})
		return
	}

	activityLog.PushBack(fmt.Sprintf("Searched items in price range: %d-%d", minPrice, maxPrice))
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Searched items in price range: %d-%d", minPrice, maxPrice),
		"status": "success",
		"data":   results,
	})
}

// Fungsi menambahkan barang baru
func addItems(c *gin.Context) {
	var newItems []Item
	if err := c.ShouldBindJSON(&newItems); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	for _, newItem := range newItems {
		var existing Item
		if err := db.Where("name = ?", newItem.Name).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Item with name %s already exists", newItem.Name)})
			return
		}
	}

	if err := db.Create(&newItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save items"})
		return
	}

	for _, item := range newItems {
		enqueueRecentItem(item)
		activityLog.PushBack(fmt.Sprintf("Added item: %s", item.Name))
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Items added successfully",
		"data":    newItems,
	})
}

// Fungsi menghapus barang berdasarkan ID
func deleteItem(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item Item
	if err := db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}

	if err := db.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete item"})
		return
	}

	activityLog.PushBack(fmt.Sprintf("Deleted item ID: %d", id))
	c.JSON(http.StatusOK, gin.H{
		"message":    "Item deleted successfully",
		"deleted_id": id,
	})
}

// Fungsi melihat barang terbaru
func getRecentItems(c *gin.Context) {
	var recentItems []Item
	for e := recentItemsQueue.Front(); e != nil; e = e.Next() {
		item, ok := e.Value.(Item)
		if ok {
			recentItems = append(recentItems, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   recentItems,
	})
}

// Fungsi melihat log aktivitas
func getActivityLog(c *gin.Context) {
	var logs []string
	for e := activityLog.Front(); e != nil; e = e.Next() {
		logs = append(logs, e.Value.(string))
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   logs,
	})
}

// Fungsi utama
func main() {
	initDB() // Inisialisasi database

	router := gin.Default()

	// Rute
	router.GET("/items/search", searchItems)
	router.GET("/items/search/price", searchItemsByPriceRange)
	router.GET("/items/search/alphabetical", searchItemsAlphabetically)
	router.POST("/items", addItems)
	router.DELETE("/items", deleteItem)
	router.GET("/items/recent", getRecentItems)
	router.GET("/activity-log", getActivityLog)

	fmt.Println("Server berjalan di http://localhost:8080")
	router.Run(":8080")
}
