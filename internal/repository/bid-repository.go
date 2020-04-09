package repository

import (
	"github.com/alexey-zayats/claim-parser/internal/database"
	"github.com/alexey-zayats/claim-parser/internal/interfaces"
	"github.com/alexey-zayats/claim-parser/internal/model"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.uber.org/dig"
)

// BidRepository ...
type BidRepository struct {
	db *sqlx.DB
}

// BidRepositoryInput ...
type BidRepositoryInput struct {
	dig.In
	DB *sqlx.DB
}

// NewBidRepository ...
func NewBidRepository(param BidRepositoryInput) interfaces.BidRepository {
	return &BidRepository{
		db: param.DB,
	}
}

// Create ...
func (r *BidRepository) Create(data *model.Bid) (int64, error) {

	var id int64

	err := database.WithTransaction(r.db, func(t database.Transaction) error {

		query := "INSERT INTO bids (" +
			"file_id, workflow_status, code, district, type, created_at, created_by, user_id, source" +
			") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"

		res, err := t.Exec(query,
			data.FileID,
			data.WorkflowStatus,
			data.Code,
			data.District,
			data.PassType,
			data.CreatedAt,
			data.CreatedBy,
			data.UserID,
			data.Source)

		if err != nil {
			return errors.Wrap(err, "unable create bid")
		}

		id, err = res.LastInsertId()
		if err != nil {
			return errors.Wrap(err, "unable get bid bids lastInsertID")
		}

		return nil
	})

	if err != nil {
		return 0, errors.Wrap(err, "transaction error")
	}

	return id, nil
}

// Update ...
func (r *BidRepository) Update(data *model.Bid) error {

	err := database.WithTransaction(r.db, func(t database.Transaction) error {

		sql := "UPDATE bids SET file_id = ?, workflow_status = ?, code = ?, " +
			"district = ?, type = ?, created_at = ?, created_by = ?, user_id = ?, source = ? " +
			"WHERE id = ?"
		_, err := t.Exec(sql,
			data.FileID,
			data.Code,
			data.District,
			data.PassType,
			data.CreatedAt,
			data.CreatedBy,
			data.UserID,
			data.ID,
			data.Source)

		if err != nil {
			return errors.Wrap(err, "unable update bids")
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "transaction error")
	}

	return nil
}

// Read ...
func (r *BidRepository) Read(id int) (*model.Bid, error) {
	var request *model.Bid

	err := r.db.Get(request, "select * from bids where id=?", id)
	if err != nil {
		return nil, errors.Wrapf(err, "unable get bids record by id %s", id)
	}

	return request, nil
}

// Delete ...
func (r *BidRepository) Delete(id int) error {

	err := database.WithTransaction(r.db, func(t database.Transaction) error {

		sql := "DELETE FROM bids WHERE id = ?"
		_, err := t.Exec(sql, id)
		if err != nil {
			return errors.Wrapf(err, "unable delete from bids by id %d", id)
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "transaction error")
	}

	return nil
}