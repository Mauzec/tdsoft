package preferences

const (
	KeyTGAPIID   = "tg.api_id"   // string
	KeyTGAPIHash = "tg.api_hash" // string
	KeyTGPhone   = "tg.phone"    // string

	KeyUIMembersMenuChat      = "ui.members_m.chat"       // string
	KeyUIMembersMenuLimit     = "ui.members_m.limit"      // string
	KeyUIMembersMenuOutput    = "ui.members_m.output"     // string
	KeyUIMembersMenuParseMsgs = "ui.members_m.parse_msgs" // bool
	KeyUIMembersMenuMsgLimit  = "ui.members_m.msg_limit"  // string
	KeyUIMembersMenuParseBio  = "ui.members_m.parse_bio"  // bool
	KeyUIMembersMenuAddInfo   = "ui.members_m.add_info"   // bool

	KeyUIMsgSearcherMenuChat     = "ui.msg_searcher_m.chat"      // string
	KeyUIMsgSearcherMenuUsername = "ui.msg_searcher_m.username"  // string
	KeyUIMsgSearcherMenuOutput   = "ui.msg_searcher_m.output"    // string
	KeyUIMsgSearcherMenuFromDate = "ui.msg_searcher_m.from_date" // string
	KeyUIMsgSearcherMenuToDate   = "ui.msg_searcher_m.to_date"   // string

	KeyUIChatStatsMenuChat   = "ui.chat_stats_m.chat"   // string
	KeyUIChatStatsMenuLimit  = "ui.chat_stats_m.limit"  // string
	KeyUIChatStatsMenuOutput = "ui.chat_stats_m.output" // string
)

const (
	DefaultUIChatStatsMenuLimit = "0"

	DefaultUIMembersMenuLimit   = "1000"
	DefaultUIMemberMenuMsgLimit = "5000"
)
