package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type mediaProductRepository struct{ db *sql.DB }

func NewMediaProductRepository(db *sql.DB) service.MediaProductRepository {
	return &mediaProductRepository{db: db}
}

func (r *mediaProductRepository) List(ctx context.Context, offset, limit int, search, modality string) ([]service.MediaCatalogProduct, int64, error) {
	conditions := []string{"1=1"}
	args := []any{}
	if search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("public_model ILIKE $%d", len(args)))
	}
	if modality != "" {
		args = append(args, modality)
		conditions = append(conditions, fmt.Sprintf("modality = $%d", len(args)))
	}
	where := strings.Join(conditions, " AND ")
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_products WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count media products: %w", err)
	}
	args = append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, public_model, modality, enabled, description, created_at, updated_at FROM media_products WHERE %s ORDER BY updated_at DESC, id DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list media products: %w", err)
	}
	defer func() { _ = rows.Close() }()
	products := []service.MediaCatalogProduct{}
	for rows.Next() {
		var product service.MediaCatalogProduct
		if err := rows.Scan(&product.ID, &product.PublicModel, &product.Modality, &product.Enabled, &product.Description, &product.CreatedAt, &product.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if err := r.loadDetails(ctx, r.db, &product); err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	return products, total, rows.Err()
}

func (r *mediaProductRepository) GetGroups(ctx context.Context, ids []int64) (map[int64]service.MediaCatalogGroup, error) {
	groups := make(map[int64]service.MediaCatalogGroup)
	if len(ids) == 0 {
		return groups, nil
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, platform, status FROM groups WHERE id = ANY($1) AND deleted_at IS NULL`, pq.Array(ids))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var group service.MediaCatalogGroup
		if err := rows.Scan(&group.ID, &group.Platform, &group.Status); err != nil {
			return nil, err
		}
		groups[group.ID] = group
	}
	return groups, rows.Err()
}

func (r *mediaProductRepository) GetByID(ctx context.Context, id int64) (*service.MediaCatalogProduct, error) {
	product := &service.MediaCatalogProduct{}
	err := r.db.QueryRowContext(ctx, `SELECT id, public_model, modality, enabled, description, created_at, updated_at FROM media_products WHERE id=$1`, id).
		Scan(&product.ID, &product.PublicModel, &product.Modality, &product.Enabled, &product.Description, &product.CreatedAt, &product.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, service.ErrMediaCatalogProductNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadDetails(ctx, r.db, product); err != nil {
		return nil, err
	}
	return product, nil
}

func (r *mediaProductRepository) GetRuntime(ctx context.Context, groupID int64, publicModel, modality string, now time.Time) (*service.MediaCatalogProduct, error) {
	var product service.MediaCatalogProduct
	err := r.db.QueryRowContext(ctx, `SELECT p.id, p.public_model, p.modality, p.enabled, p.description, p.created_at, p.updated_at FROM media_products p JOIN media_product_group_bindings b ON b.product_id=p.id WHERE b.group_id=$1 AND p.public_model=$2 AND p.modality=$3 AND p.enabled=TRUE`, groupID, publicModel, modality).
		Scan(&product.ID, &product.PublicModel, &product.Modality, &product.Enabled, &product.Description, &product.CreatedAt, &product.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrMediaCatalogProductNotFound
	}
	if err != nil {
		return nil, err
	}
	if err = r.loadDetails(ctx, r.db, &product); err != nil {
		return nil, err
	}
	prices := product.Prices[:0]
	for _, price := range product.Prices {
		if price.Enabled {
			prices = append(prices, price)
		}
	}
	product.Prices = prices
	offers := product.Offers[:0]
	for _, offer := range product.Offers {
		if offer.Enabled && offer.ExpiresAt.After(now) {
			offers = append(offers, offer)
		}
	}
	product.Offers = offers
	if len(product.Prices) == 0 || len(product.Offers) == 0 {
		return nil, service.ErrMediaCatalogProductNotFound
	}
	return &product, nil
}

func (r *mediaProductRepository) ListRuntimeModels(ctx context.Context, groupID int64, now time.Time) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT p.public_model FROM media_products p JOIN media_product_group_bindings b ON b.product_id=p.id WHERE b.group_id=$1 AND p.enabled=TRUE AND EXISTS (SELECT 1 FROM media_product_prices pp WHERE pp.product_id=p.id AND pp.enabled=TRUE) AND EXISTS (SELECT 1 FROM media_offers o JOIN groups g ON g.id=o.source_group_id AND g.deleted_at IS NULL AND g.status=$2 WHERE o.product_id=p.id AND o.enabled=TRUE AND o.expires_at>$3) ORDER BY p.public_model`, groupID, service.StatusActive, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	models := []string{}
	for rows.Next() {
		var model string
		if err := rows.Scan(&model); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, rows.Err()
}

func (r *mediaProductRepository) ListRuntimeModelModalities(ctx context.Context, groupID int64, now time.Time) ([]service.MediaRuntimeModel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT p.public_model, p.modality FROM media_products p JOIN media_product_group_bindings b ON b.product_id=p.id WHERE b.group_id=$1 AND p.enabled=TRUE AND EXISTS (SELECT 1 FROM media_product_prices pp WHERE pp.product_id=p.id AND pp.enabled=TRUE) AND EXISTS (SELECT 1 FROM media_offers o JOIN groups g ON g.id=o.source_group_id AND g.deleted_at IS NULL AND g.status=$2 WHERE o.product_id=p.id AND o.enabled=TRUE AND o.expires_at>$3) ORDER BY p.public_model, p.modality`, groupID, service.StatusActive, now)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	models := []service.MediaRuntimeModel{}
	for rows.Next() {
		var model service.MediaRuntimeModel
		if err := rows.Scan(&model.Model, &model.Modality); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, rows.Err()
}

func (r *mediaProductRepository) Create(ctx context.Context, product *service.MediaCatalogProduct) error {
	return r.save(ctx, product, true)
}

func (r *mediaProductRepository) Update(ctx context.Context, product *service.MediaCatalogProduct) error {
	return r.save(ctx, product, false)
}

func (r *mediaProductRepository) Disable(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE media_products SET enabled=FALSE, updated_at=NOW() WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return service.ErrMediaCatalogProductNotFound
	}
	return nil
}

func (r *mediaProductRepository) save(ctx context.Context, product *service.MediaCatalogProduct, create bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if create {
		err = tx.QueryRowContext(ctx, `INSERT INTO media_products(public_model, modality, enabled, description) VALUES($1,$2,$3,$4) RETURNING id, created_at, updated_at`, product.PublicModel, product.Modality, product.Enabled, product.Description).
			Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)
	} else {
		result, updateErr := tx.ExecContext(ctx, `UPDATE media_products SET public_model=$1, modality=$2, enabled=$3, description=$4, updated_at=NOW() WHERE id=$5`, product.PublicModel, product.Modality, product.Enabled, product.Description, product.ID)
		err = updateErr
		if err == nil {
			if n, _ := result.RowsAffected(); n == 0 {
				return service.ErrMediaCatalogProductNotFound
			}
			_, err = tx.ExecContext(ctx, `DELETE FROM media_product_group_bindings WHERE product_id=$1`, product.ID)
			if err == nil {
				_, err = tx.ExecContext(ctx, `DELETE FROM media_product_prices WHERE product_id=$1`, product.ID)
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE media_offers SET enabled=FALSE, updated_at=NOW() WHERE product_id=$1`, product.ID)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("save media product: %w", err)
	}
	for _, groupID := range product.GroupIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_product_group_bindings(product_id, group_id) VALUES($1,$2)`, product.ID, groupID); err != nil {
			return err
		}
	}
	for _, price := range product.Prices {
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_product_prices(product_id, operation, spec_key, unit_price_usd, currency, version, enabled) VALUES($1,$2,$3,$4,$5,$6,$7)`, product.ID, price.Operation, price.SpecKey, price.UnitPriceUSD, price.Currency, price.Version, price.Enabled); err != nil {
			return err
		}
	}
	for _, offer := range product.Offers {
		operations, err := json.Marshal(offer.Operations)
		if err != nil {
			return err
		}
		capabilities, err := json.Marshal(offer.Capabilities)
		if err != nil {
			return err
		}
		costRules, err := json.Marshal(offer.CostRules)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_offers(product_id, provider, source_group_id, upstream_model, enabled, priority, operations, capabilities, cost_rules, cost_source, cost_version, verified_at, expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (product_id, provider, source_group_id, upstream_model) DO UPDATE SET enabled=EXCLUDED.enabled, priority=EXCLUDED.priority, operations=EXCLUDED.operations, capabilities=EXCLUDED.capabilities, cost_rules=EXCLUDED.cost_rules, cost_source=EXCLUDED.cost_source, cost_version=EXCLUDED.cost_version, verified_at=EXCLUDED.verified_at, expires_at=EXCLUDED.expires_at, updated_at=NOW()`, product.ID, offer.Provider, offer.SourceGroupID, offer.UpstreamModel, offer.Enabled, offer.Priority, operations, capabilities, costRules, offer.CostSource, offer.CostVersion, offer.VerifiedAt, offer.ExpiresAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *mediaProductRepository) loadDetails(ctx context.Context, q queryer, product *service.MediaCatalogProduct) error {
	product.GroupIDs = []int64{}
	product.Prices = []service.MediaCatalogPrice{}
	product.Offers = []service.MediaCatalogOffer{}
	rows, err := q.QueryContext(ctx, `SELECT group_id FROM media_product_group_bindings WHERE product_id=$1 ORDER BY group_id`, product.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
		product.GroupIDs = append(product.GroupIDs, id)
	}
	_ = rows.Close()
	rows, err = q.QueryContext(ctx, `SELECT id, operation, spec_key, unit_price_usd, currency, version, enabled FROM media_product_prices WHERE product_id=$1 ORDER BY id`, product.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var p service.MediaCatalogPrice
		if err := rows.Scan(&p.ID, &p.Operation, &p.SpecKey, &p.UnitPriceUSD, &p.Currency, &p.Version, &p.Enabled); err != nil {
			_ = rows.Close()
			return err
		}
		product.Prices = append(product.Prices, p)
	}
	_ = rows.Close()
	rows, err = q.QueryContext(ctx, `SELECT id, provider, source_group_id, upstream_model, enabled, priority, operations, capabilities, cost_rules, cost_source, cost_version, verified_at, expires_at FROM media_offers WHERE product_id=$1 ORDER BY priority,id`, product.ID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var o service.MediaCatalogOffer
		var operationsJSON, capabilitiesJSON, costRulesJSON []byte
		if err := rows.Scan(&o.ID, &o.Provider, &o.SourceGroupID, &o.UpstreamModel, &o.Enabled, &o.Priority, &operationsJSON, &capabilitiesJSON, &costRulesJSON, &o.CostSource, &o.CostVersion, &o.VerifiedAt, &o.ExpiresAt); err != nil {
			return err
		}
		if err := json.Unmarshal(operationsJSON, &o.Operations); err != nil {
			return err
		}
		if err := json.Unmarshal(capabilitiesJSON, &o.Capabilities); err != nil {
			return err
		}
		if err := json.Unmarshal(costRulesJSON, &o.CostRules); err != nil {
			return err
		}
		product.Offers = append(product.Offers, o)
	}
	return rows.Err()
}
