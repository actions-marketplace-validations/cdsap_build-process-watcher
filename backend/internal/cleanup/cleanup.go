package cleanup

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cdsap/build-process-watcher/backend/internal/auth"
	"github.com/cdsap/build-process-watcher/backend/internal/storage"
)

const (
	// BuildTimeout is the timeout for marking builds as finished (5 minutes)
	BuildTimeout = 5 * time.Minute
	// DataRetentionPeriod is the period for retaining data (3 hours)
	DataRetentionPeriod = 3 * time.Hour
)

// Service handles cleanup operations
type Service struct {
	storage *storage.Client
}

// NewService creates a new cleanup service
func NewService(storageClient *storage.Client) *Service {
	return &Service{
		storage: storageClient,
	}
}

// StartStaleRunCleanup starts the background cleanup routine for stale runs
func (s *Service) StartStaleRunCleanup() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()

	log.Printf("🧹 Started background cleanup routine for stale runs (checking every 2 minutes)")

	for {
		select {
		case <-ticker.C:
			log.Printf("🧹 Running cleanup check for stale runs...")
			
			staleRuns, err := s.storage.FindStaleRuns(BuildTimeout)
			if err != nil {
				log.Printf("❌ Error finding stale runs: %v", err)
				continue
			}

			log.Printf("🧹 Found %d stale runs", len(staleRuns))

			// Mark stale runs as finished
			for _, runID := range staleRuns {
				err := s.storage.MarkRunAsFinished(runID)
				if err != nil {
					log.Printf("❌ Error cleaning up stale run %s: %v", runID, err)
				} else {
					log.Printf("✅ Successfully marked stale run %s as finished", runID)
				}
			}

			if len(staleRuns) > 0 {
				log.Printf("🧹 Cleaned up %d stale runs", len(staleRuns))
			} else {
				log.Printf("🧹 No stale runs found")
			}
		}
	}
}

// StartDataRetentionCleanup starts the background cleanup routine for old data
func (s *Service) StartDataRetentionCleanup() {
	ticker := time.NewTicker(30 * time.Minute) // Check every 30 minutes
	defer ticker.Stop()
	
	log.Printf("🗑️ Started data retention cleanup routine (checking every 30 minutes, retention: %v)", DataRetentionPeriod)
	
	for {
		select {
		case <-ticker.C:
			log.Printf("🗑️ Running data retention cleanup...")
			
			deletedRuns, err := s.storage.DeleteOldRuns(DataRetentionPeriod)
			if err != nil {
				log.Printf("❌ Error deleting old runs: %v", err)
				continue
			}
			
			if len(deletedRuns) > 0 {
				log.Printf("🗑️ Cleaned up %d old runs (older than %v)", len(deletedRuns), DataRetentionPeriod)
			} else {
				log.Printf("🗑️ No old data to clean up")
			}
		}
	}
}

// HandleManualStaleCleanup handles manual cleanup of stale runs (admin only)
func (s *Service) HandleManualStaleCleanup(w http.ResponseWriter, r *http.Request) {
	log.Printf("cleanupStaleHandler called with method: %s", r.Method)
	
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Secret")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Require admin authentication
	if !auth.RequireAdminAuth(r) {
		log.Printf("⚠️  Unauthorized cleanup attempt from %s", r.RemoteAddr)
		http.Error(w, "Unauthorized - admin secret required", http.StatusUnauthorized)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	log.Printf("🧹 Manual cleanup triggered...")
	
	staleRuns, err := s.storage.FindStaleRuns(BuildTimeout)
	if err != nil {
		log.Printf("❌ Error finding stale runs: %v", err)
		http.Error(w, fmt.Sprintf("Error finding stale runs: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("🧹 Found %d stale runs", len(staleRuns))

	// Mark stale runs as finished
	var cleanedRuns []string
	for _, runID := range staleRuns {
		err := s.storage.MarkRunAsFinished(runID)
		if err != nil {
			log.Printf("❌ Error cleaning up stale run %s: %v", runID, err)
		} else {
			log.Printf("✅ Successfully marked stale run %s as finished", runID)
			cleanedRuns = append(cleanedRuns, runID)
		}
	}

	response := map[string]interface{}{
		"success":       true,
		"total_checked": len(staleRuns),
		"stale_found":   len(staleRuns),
		"cleaned_up":    len(cleanedRuns),
		"cleaned_runs":  cleanedRuns,
	}

	if len(staleRuns) > 0 {
		log.Printf("🧹 Manual cleanup completed: cleaned up %d stale runs", len(cleanedRuns))
	} else {
		log.Printf("🧹 Manual cleanup completed: no stale runs found")
	}

	json.NewEncoder(w).Encode(response)
}

// HandleManualDataRetentionCleanup handles manual cleanup of old data (admin only)
func (s *Service) HandleManualDataRetentionCleanup(w http.ResponseWriter, r *http.Request) {
	log.Printf("cleanupOldDataHandler called with method: %s", r.Method)
	
	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-Secret")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Require admin authentication
	if !auth.RequireAdminAuth(r) {
		log.Printf("⚠️  Unauthorized cleanup attempt from %s", r.RemoteAddr)
		http.Error(w, "Unauthorized - admin secret required", http.StatusUnauthorized)
		return
	}

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	log.Printf("🗑️ Manual data retention cleanup triggered...")
	
	deletedRuns, err := s.storage.DeleteOldRuns(DataRetentionPeriod)
	if err != nil {
		log.Printf("❌ Error deleting old runs: %v", err)
		http.Error(w, fmt.Sprintf("Error deleting runs: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":       true,
		"cutoff_time":   time.Now().Add(-DataRetentionPeriod),
		"deleted_count": len(deletedRuns),
		"deleted_runs":  deletedRuns,
	}

	if len(deletedRuns) > 0 {
		log.Printf("🗑️ Manual cleanup completed: deleted %d old runs (older than %v)", len(deletedRuns), DataRetentionPeriod)
	} else {
		log.Printf("🗑️ Manual cleanup completed: no old data to clean up")
	}

	json.NewEncoder(w).Encode(response)
}

