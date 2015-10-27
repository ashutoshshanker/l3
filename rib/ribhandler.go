package main
import ("ribd")

type RouteServiceHandler struct {
}

func (m RouteServiceHandler) CreateV4Route( destNet         ribd.Int, 
                                            prefixLen       ribd.Int, 
                                            nextHop         ribd.Int, 
                                            nextHopIfIndex  ribd.Int,
                                            metric          ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received create route request")
    return 0, nil
}

func (m RouteServiceHandler) DeleteV4Route( destNet         ribd.Int, 
                                            prefixLen       ribd.Int, 
                                            nextHop         ribd.Int, 
                                            nextHopIfIndex  ribd.Int) (rc ribd.Int, err error) {
    logger.Println("Received Route Delete request")
    return 0, nil
}

func NewRouteServiceHandler () *RouteServiceHandler {
    return &RouteServiceHandler{}
}
