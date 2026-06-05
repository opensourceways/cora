package view

// builtinViews contains the default display configurations shipped with cora.
// User entries in views.yaml override individual keys here (whole-config replacement,
// not column merging).
var builtinViews = map[string]map[string]ViewConfig{
	"gitcode": {
		"issues/get": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No."},
				{Field: "title", Label: "Title", Truncate: 120},
				{Field: "state", Label: "State", Colorize: true},
				{Field: "user.login", Label: "Author"},
				{Field: "assignees", Label: "Assignees", Format: FormatJSON},
				{Field: "labels", Label: "Labels", Format: FormatJSON},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "updated_at", Label: "Updated", Format: FormatDate},
				{Field: "body", Label: "Description", Format: FormatMultiline, Truncate: 600},
			},
		},
		"issues/list": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No.", Width: 6},
				{Field: "title", Label: "Title", Truncate: 50, Width: 52},
				{Field: "state", Label: "State", Width: 8, Colorize: true},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"issues/comments": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 8},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "body", Label: "Body", Format: FormatMultiline, Truncate: 160, Width: 56},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
				{Field: "updated_at", Label: "Updated", Format: FormatDate, Width: 12},
			},
		},
		"users/list-user": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 22},
				{Field: "login", Label: "Login", Width: 16},
				{Field: "name", Label: "Name", Width: 16},
				{Field: "email", Label: "Email", Width: 24},
				{Field: "company", Label: "Company", Width: 14},
				{Field: "location", Label: "Location", Width: 14},
				{Field: "followers", Label: "Followers", Width: 8},
				{Field: "following", Label: "Following", Width: 8},
				{Field: "type", Label: "Type", Width: 8},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"users/get": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 22},
				{Field: "login", Label: "Login", Width: 16},
				{Field: "name", Label: "Name", Width: 16},
				{Field: "email", Label: "Email", Width: 24},
				{Field: "company", Label: "Company", Width: 14},
				{Field: "location", Label: "Location", Width: 14},
				{Field: "followers", Label: "Followers", Width: 8},
				{Field: "following", Label: "Following", Width: 8},
				{Field: "type", Label: "Type", Width: 8},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"repos/get": {
			Columns: []ViewColumn{
				{Field: "full_name", Label: "Repo"},
				{Field: "description", Label: "Description", Truncate: 80},
				{Field: "stargazers_count", Label: "Stars"},
				{Field: "forks_count", Label: "Forks"},
				{Field: "language", Label: "Language"},
				{Field: "license.name", Label: "License"},
				{Field: "topics", Label: "Topics", Format: FormatJSON},
				{Field: "created_at", Label: "Created", Format: FormatDate},
			},
		},
		"repos/list": {
			Columns: []ViewColumn{
				{Field: "full_name", Label: "Repo", Width: 32},
				{Field: "description", Label: "Description", Truncate: 40, Width: 42},
				{Field: "stargazers_count", Label: "Stars", Width: 8},
				{Field: "language", Label: "Language", Width: 12},
			},
		},
		"pulls/list": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No.", Width: 6},
				{Field: "title", Label: "Title", Truncate: 50, Width: 52},
				{Field: "state", Label: "State", Width: 8, Colorize: true},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "head.label", Label: "Branch", Width: 24},
				{Field: "html_url", Label: "URL", Truncate: 50, Width: 52},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"pulls/get": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No."},
				{Field: "title", Label: "Title", Truncate: 120},
				{Field: "state", Label: "State", Colorize: true},
				{Field: "user.login", Label: "Author"},
				{Field: "head.label", Label: "From Branch"},
				{Field: "base.label", Label: "Into Branch"},
				{Field: "html_url", Label: "URL"},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "body", Label: "Description", Format: FormatMultiline, Truncate: 600},
			},
		},
	},

	"github": {
		"issues/get": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No."},
				{Field: "title", Label: "Title", Truncate: 120},
				{Field: "state", Label: "State", Colorize: true},
				{Field: "user.login", Label: "Author"},
				{Field: "assignees", Label: "Assignees", Format: FormatJSON},
				{Field: "labels", Label: "Labels", Format: FormatJSON},
				{Field: "html_url", Label: "URL"},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "updated_at", Label: "Updated", Format: FormatDate},
				{Field: "body", Label: "Description", Format: FormatMultiline, Truncate: 600},
			},
		},
		"issues/list": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No.", Width: 6},
				{Field: "title", Label: "Title", Truncate: 50, Width: 52},
				{Field: "state", Label: "State", Width: 8, Colorize: true},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"issues/comments": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 8},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "body", Label: "Body", Format: FormatMultiline, Truncate: 160, Width: 56},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
				{Field: "updated_at", Label: "Updated", Format: FormatDate, Width: 12},
			},
		},
		"repos/get": {
			Columns: []ViewColumn{
				{Field: "full_name", Label: "Repo"},
				{Field: "description", Label: "Description", Truncate: 120},
				{Field: "stargazers_count", Label: "Stars"},
				{Field: "forks_count", Label: "Forks"},
				{Field: "open_issues_count", Label: "Open Issues"},
				{Field: "language", Label: "Language"},
				{Field: "license.name", Label: "License"},
				{Field: "default_branch", Label: "Default Branch"},
				{Field: "topics", Label: "Topics", Format: FormatJSON},
				{Field: "html_url", Label: "URL"},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "pushed_at", Label: "Pushed", Format: FormatDate},
			},
		},
		"repos/list": {
			Columns: []ViewColumn{
				{Field: "full_name", Label: "Repo", Width: 32},
				{Field: "description", Label: "Description", Truncate: 40, Width: 42},
				{Field: "stargazers_count", Label: "Stars", Width: 8},
				{Field: "language", Label: "Language", Width: 12},
				{Field: "updated_at", Label: "Updated", Format: FormatDate, Width: 12},
			},
		},
		"pulls/list": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No.", Width: 6},
				{Field: "title", Label: "Title", Truncate: 50, Width: 52},
				{Field: "state", Label: "State", Width: 8, Colorize: true},
				{Field: "user.login", Label: "Author", Width: 18},
				{Field: "head.label", Label: "Branch", Width: 24},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"pulls/get": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No."},
				{Field: "title", Label: "Title", Truncate: 120},
				{Field: "state", Label: "State", Colorize: true},
				{Field: "user.login", Label: "Author"},
				{Field: "head.label", Label: "From Branch"},
				{Field: "base.label", Label: "Into Branch"},
				{Field: "merged", Label: "Merged"},
				{Field: "html_url", Label: "URL"},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "body", Label: "Description", Format: FormatMultiline, Truncate: 600},
			},
		},
		"users/list-user": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 22},
				{Field: "login", Label: "Login", Width: 16},
				{Field: "name", Label: "Name", Width: 16},
				{Field: "email", Label: "Email", Width: 24},
				{Field: "company", Label: "Company", Width: 14},
				{Field: "location", Label: "Location", Width: 14},
				{Field: "followers", Label: "Followers", Width: 8},
				{Field: "following", Label: "Following", Width: 8},
				{Field: "type", Label: "Type", Width: 8},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"users/get": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 22},
				{Field: "login", Label: "Login", Width: 16},
				{Field: "name", Label: "Name", Width: 16},
				{Field: "email", Label: "Email", Width: 24},
				{Field: "company", Label: "Company", Width: 14},
				{Field: "location", Label: "Location", Width: 14},
				{Field: "followers", Label: "Followers", Width: 8},
				{Field: "following", Label: "Following", Width: 8},
				{Field: "type", Label: "Type", Width: 8},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
	},

	"forum": {
		"topics/list": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 8},
				{Field: "title", Label: "Title", Truncate: 60, Width: 62},
				{Field: "posts_count", Label: "Posts", Width: 8},
				{Field: "reply_count", Label: "Replies", Width: 8},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
		"topics/get": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID"},
				{Field: "title", Label: "Title"},
				{Field: "posts_count", Label: "Posts"},
				{Field: "reply_count", Label: "Replies"},
				{Field: "created_at", Label: "Created", Format: FormatDate},
				{Field: "category_id", Label: "Category"},
			},
		},
		"posts/list": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 8},
				{Field: "username", Label: "Author", Width: 18},
				{Field: "cooked", Label: "Content", Format: FormatMultiline, Truncate: 200, Width: 60},
				{Field: "created_at", Label: "Created", Format: FormatDate, Width: 12},
			},
		},
	},

	"etherpad": {
		"pads/list": {
			Columns: []ViewColumn{
				{Field: "padID", Label: "Pad ID"},
			},
		},
	},
	"jenkins": {
		"jobs/list": {
			RootField: "jobs",
			Columns: []ViewColumn{
				{Field: "name", Label: "Name", Width: 40},
				{Field: "color", Label: "Status", Width: 12, Colorize: true},
				{Field: "url", Label: "URL", Truncate: 60, Width: 62},
			},
		},
		"jobs/get": {
			Columns: []ViewColumn{
				{Field: "name", Label: "Name"},
				{Field: "description", Label: "Description", Truncate: 120},
				{Field: "color", Label: "Status", Colorize: true},
				{Field: "buildable", Label: "Buildable"},
				{Field: "inQueue", Label: "In Queue"},
				{Field: "nextBuildNumber", Label: "Next Build"},
				{Field: "url", Label: "URL"},
			},
		},
		"jobs/build": {
			Columns: []ViewColumn{
				{Field: "id", Label: "Queue ID"},
				{Field: "task.name", Label: "Project"},
				{Field: "why", Label: "Why"},
				{Field: "inQueueSince", Label: "Waiting (ms)"},
			},
		},
		"builds/get": {
			Columns: []ViewColumn{
				{Field: "number", Label: "No."},
				{Field: "displayName", Label: "Name"},
				{Field: "result", Label: "Result", Colorize: true},
				{Field: "building", Label: "Building", Colorize: true},
				{Field: "duration", Label: "Duration (ms)"},
				{Field: "timestamp", Label: "Timestamp", Format: FormatDate},
				{Field: "url", Label: "URL"},
				{Field: "artifacts", Label: "Artifacts", Format: FormatJSON},
			},
		},
		"builds/progressive-log": {
			Columns: []ViewColumn{
				{Field: "text", Label: "Log", Format: FormatMultiline, Truncate: 2000},
				{Field: "hasMore", Label: "Has More"},
				{Field: "offset", Label: "Offset"},
				{Field: "size", Label: "Size"},
			},
		},
		"queue/list": {
			RootField: "items",
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 8},
				{Field: "task.name", Label: "Task", Width: 40},
				{Field: "why", Label: "Why", Width: 30},
				{Field: "inQueueSince", Label: "Waiting (ms)", Width: 14},
				{Field: "blocked", Label: "Blocked", Width: 8},
				{Field: "stuck", Label: "Stuck", Width: 6},
			},
		},
	},
	"eur": {
		"builds/get": {
			Columns: []ViewColumn{
				{Field: "id", Label: "ID", Width: 10},
				{Field: "state", Label: "State", Colorize: true},
				{Field: "chroot", Label: "Chroot", Width: 20},
				{Field: "source_package", Label: "Source Package", Width: 18},
				{Field: "submitted_on", Label: "Submitted", Format: FormatDate},
				{Field: "started_on", Label: "Started", Format: FormatDate},
				{Field: "ended_on", Label: "Ended", Format: FormatDate},
			},
		},
		"packages/get": {
			Columns: []ViewColumn{
				{Field: "name", Label: "Package", Width: 24},
				{Field: "source_type", Label: "Source Type", Width: 12},
				{Field: "ownername", Label: "Owner", Width: 16},
				{Field: "projectname", Label: "Project", Width: 20},
			},
		},
	},
}
