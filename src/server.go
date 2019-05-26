package filerelay

import (
	"fmt"
)

//
type Server struct {
	maxRoutines int
}

//
func (s *Server) handleConn() error {
	fmt.Println("Nothing here...")
	return nil
}
