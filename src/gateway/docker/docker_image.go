package docker

import (
	"time"

	apsql "gateway/sql"
)

type DockerImage struct {
	ID        int64      `json:"id,omitempty" path:"id"`
	CreatedAt *time.Time `json:"-" db:"created_at"`
	UpdatedAt *time.Time `json:"-" db:"updated_at"`
	ClientID  string     `json:"client_id" db:"client_id"`
	Name      string     `json:"name"`
}

func (d *DockerImage) AllStale(db *apsql.DB, cutoff time.Duration) ([]*DockerImage, error) {
	images := []*DockerImage{}
	err := db.Select(&images, db.SQL("docker_images/all_stale"), d.ClientID, time.Now().UTC().Add(-cutoff))
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (d *DockerImage) Find(db *apsql.DB) (*DockerImage, error) {
	image := DockerImage{}
	err := db.Get(&image, db.SQL("docker_images/find"), d.Name, d.ClientID)
	return &image, err
}

func (d *DockerImage) Delete(tx *apsql.Tx) error {
	return tx.DeleteOne(tx.SQL("docker_images/delete"), d.ID)
}

func (d *DockerImage) Insert(tx *apsql.Tx) error {
	var err error
	d.ID, err = tx.InsertOne(tx.SQL("docker_images/insert"), d.Name, d.ClientID)
	return err
}

func (d *DockerImage) Update(tx *apsql.Tx) error {
	return tx.UpdateOne(tx.SQL("docker_images/update"), d.Name, d.ClientID, d.ID)
}
