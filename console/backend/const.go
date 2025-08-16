package backend

import (
	"net/http"

	"github.com/Alonza0314/free-ran-ue/util"
)

const (
	APPLICATION_JSON = "application/json"

	API_GNB_INFO        = util.GNB_API_PREFIX + "/info"
	API_GNB_INFO_METHOD = http.MethodGet

	API_GNB_UE_NRDC_MODIFY        = util.GNB_API_PREFIX + "/ue/nrdc"
	API_GNB_UE_NRDC_MODIFY_METHOD = http.MethodPost
)
