package model

const (
	Leader    = 0 //领导者
	Follower  = 1 //跟随者
	Candidate = 2 //候选人

	//command
	Canvass        = 0 //选举
	Heart          = 1 //心跳
	LogReplication = 2 //日志复制
)

type Node struct {
	Name        string //节点名称
	Address     string //本节点IP端口
	Role        int    //当前角色
	State       int    //节点状态
	Leader      string
	Time        int  //选举次数 (Term)
	CanvassFlag bool //选票状态  true 还有未投选票, false 已投选票
	CanvassNum  int
}

type CommandMsg struct {
	Command     int
	Msg         string
	LogCommand  RequestBody
	Node        Node
	CanvassFlag bool
	Err         error
}
