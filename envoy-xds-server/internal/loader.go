package internal

func LoadUserServer() []*UserServer {
	return []*UserServer{
		{
			UserId:      "1",
			Deployment:  "server1",
			ServiceName: "server1",
			ServicePort: 8080,
		},
		{
			UserId:      "2",
			Deployment:  "server2",
			ServiceName: "server2",
			ServicePort: 7070,
		},
	}
}
