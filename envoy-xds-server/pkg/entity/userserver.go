package entity

type UserServer struct {
	UserId      string
	ServiceName string
	ServicePort uint32
}

type UserServerSync struct {
	UserId    string
	Entry     *UserServer
	IsRemoved bool
}
