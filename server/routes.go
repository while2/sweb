package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/mijia/sweb/log"
)

const (
	kAssetsReverseKey = "_!#assets_"
)

func (s *Server) EnableExtraAssetsMapping(assetsMapping map[string]string) {
	s.extraAssetsMapping = assetsMapping
}

func (s *Server) Reverse(name string, params ...interface{}) string {
	path, ok := s.namedRoutes[name]
	if !ok {
		log.Warnf("Server routes reverse failed, cannot find named routes %q", name)
		return "/no_such_named_routes_defined"
	}
	if len(params) == 0 || path == "/" {
		return path
	}
	strParams := make([]string, len(params))
	for i, param := range params {
		strParams[i] = fmt.Sprint(param)
	}
	parts := strings.Split(path, "/")[1:]
	paramIndex := 0
	for i, part := range parts {
		if part[0] == ':' || part[0] == '*' {
			if paramIndex < len(strParams) {
				parts[i] = strParams[paramIndex]
				paramIndex++
			}
		}
	}
	return httprouter.CleanPath("/" + strings.Join(parts, "/"))
}

func (s *Server) Assets(path string) string {
	if asset, ok := s.extraAssetsMapping[path]; ok {
		path = asset
	}
	return s.Reverse(kAssetsReverseKey, path)
}

func (s *Server) Files(path string, root http.FileSystem) {
	s.router.ServeFiles(path, root)
	s.namedRoutes[kAssetsReverseKey] = path
}
