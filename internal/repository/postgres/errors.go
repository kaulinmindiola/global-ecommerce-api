package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/domain"
)

// PostgreSQL error codes relevant to our domain.
// Reference: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	pgErrUniqueViolation     = "23505" // duplicate key value violates unique constraint
	pgErrForeignKeyViolation = "23503" // violates foreign key constraint
	pgErrNotNullViolation    = "23502" // null value in column violates not-null constraint
	pgErrCheckViolation      = "23514" // new row violates check constraint
)

// mapPgError translates low-level pgconn errors into domain-level errors.
// This keeps the repository layer from leaking infrastructure details into
// the service layer — callers only need to handle domain.Err* sentinels.
func mapPgError(err error) error {
	var pgErr *pgconn.PgError

	// Si el error no es de tipo pgconn (Postgres), lo devolvemos tal cual.
	if !errors.As(err, &pgErr) {
		return err
	}

	// Mapeamos el código de error de Postgres a un error de nuestro Dominio.
	switch pgErr.Code {
	case pgErrUniqueViolation:
		return domain.ErrConflict
	case pgErrForeignKeyViolation:
		return domain.ErrInvalidReference
	case pgErrNotNullViolation, pgErrCheckViolation:
		return domain.ErrInvalidInput
	default:
		return err
	}
}
