package port

import (
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/model"
)

type SKURepo port.Repo[*model.SKU]
