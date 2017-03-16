package service

import (
	"time"

	"gateway/config"
	"gateway/docker"
	"gateway/logreport"
	"gateway/sql"
)

func DockerImageDeletionService(conf config.Configuration, db *sql.DB) {
	if !conf.Jobs || !docker.Available() {
		return
	}

	report := func(message string, err error) {
		logreport.Printf("[docker] %v: %v", message, err)
	}

	timeout := time.Duration(conf.Docker.ImageIdleTimeout)
	deleteTicker := time.NewTicker(timeout * time.Hour)
	clientID, err := docker.DockerClientID()
	if err != nil {
		report("getting client id", err)
		return
	}
	go func() {
		for _ = range deleteTicker.C {
			image := docker.DockerImage{
				ClientID: clientID,
			}
			images, err := image.AllStale(db, timeout*time.Hour)
			if err != nil {
				report("looking up stale images", err)
				continue
			}
			for _, image := range images {
				err := docker.DeleteImage(image.Name)
				if err != nil {
					report("deleting image", err)
				}
				err = db.DoInTransaction(func(tx *sql.Tx) error {
					return image.Delete(tx)
				})
				if err != nil {
					report("deleting image db entry", err)
				}
			}
		}
	}()
}
