// Package ui provides web UI handlers and templates for the login page,
// user management, and OIDC request/response log viewer.
package ui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"oauth-tester/internal/store"
)

type PageHandlers struct {
	Store store.Store
}

var usersTmpl = template.Must(template.New("users").Parse(`<!DOCTYPE html>
<html>
<head>
<title>Users - oauth-tester</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
  nav { background: #1a1a2e; padding: 12px 24px; display: flex; gap: 16px; }
  nav a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  nav a:hover { color: #fff; }
  nav a.active { color: #fff; font-weight: bold; }
  .container { max-width: 900px; margin: 24px auto; padding: 0 24px; }
  h1 { font-size: 24px; }
  table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
  th, td { padding: 10px 14px; text-align: left; border-bottom: 1px solid #eee; }
  th { background: #f8f8f8; font-size: 13px; text-transform: uppercase; color: #666; }
  td input { border: 1px solid transparent; padding: 4px 6px; width: 100%; box-sizing: border-box; background: transparent; }
  td input:focus { border-color: #1a1a2e; outline: none; background: #fff; }
  .actions { display: flex; gap: 8px; margin-bottom: 16px; }
  .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .btn-add { background: #1a1a2e; color: #fff; }
  .btn-add:hover { background: #16213e; }
  .btn-del { background: #d32f2f; color: #fff; font-size: 12px; padding: 4px 10px; }
  .btn-del:hover { background: #b71c1c; }
</style>
</head>
<body>
<nav>
  <a href="/ui/users" class="active">Users</a>
  <a href="/ui/groups">Groups</a>
  <a href="/ui/logs">Logs</a>
</nav>
<div class="container">
  <h1>Users</h1>
  <div class="actions">
    <button class="btn btn-add" onclick="addUser()">Add User</button>
  </div>
  <table>
    <thead>
      <tr><th>Username</th><th>Password</th><th>Email</th><th>Name</th><th></th></tr>
    </thead>
    <tbody id="user-table"></tbody>
  </table>
</div>
<script>
function load() {
  fetch('/api/users').then(r => r.json()).then(users => {
    const tbody = document.getElementById('user-table');
    tbody.innerHTML = '';
    users.forEach(u => {
      const tr = document.createElement('tr');
      tr.innerHTML =
        '<td><input value="' + esc(u.Username) + '" disabled></td>' +
        '<td><input value="' + esc(u.Password) + '" onchange="upd(\'' + esc(u.Username) + '\', this.parentNode.parentNode)"></td>' +
        '<td><input value="' + esc(u.Email) + '" onchange="upd(\'' + esc(u.Username) + '\', this.parentNode.parentNode)"></td>' +
        '<td><input value="' + esc(u.Name) + '" onchange="upd(\'' + esc(u.Username) + '\', this.parentNode.parentNode)"></td>' +
        '<td><button class="btn btn-del" onclick="del(\'' + esc(u.Username) + '\')">Delete</button></td>';
      tbody.appendChild(tr);
    });
  });
}
function esc(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML.replace(/'/g, "&#39;").replace(/"/g, '&quot;'); }
function addUser() {
  const name = prompt('Username:');
  if (!name) return;
  fetch('/api/users', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({Username: name, Password: '', Email: '', Name: ''}) })
    .then(() => load());
}
function upd(username, row) {
  const inputs = row.querySelectorAll('input');
  fetch('/api/users/' + encodeURIComponent(username), {
    method: 'PUT', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ Password: inputs[1].value, Email: inputs[2].value, Name: inputs[3].value })
  }).then(() => load());
}
function del(username) {
  if (!confirm('Delete ' + username + '?')) return;
  fetch('/api/users/' + encodeURIComponent(username), { method: 'DELETE' }).then(() => load());
}
load();
</script>
</body>
</html>`))

var logsTmpl = template.Must(template.New("logs").Parse(`<!DOCTYPE html>
<html>
<head>
<title>Logs - oauth-tester</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
  nav { background: #1a1a2e; padding: 12px 24px; display: flex; gap: 16px; }
  nav a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  nav a:hover { color: #fff; }
  nav a.active { color: #fff; font-weight: bold; }
  .container { max-width: 1100px; margin: 24px auto; padding: 0 24px; }
  h1 { font-size: 24px; }
  .actions { margin-bottom: 16px; }
  .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .btn-clear { background: #d32f2f; color: #fff; }
  .btn-clear:hover { background: #b71c1c; }
  .log-entry { background: #fff; border-radius: 8px; padding: 16px; margin-bottom: 16px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
  .log-header { display: flex; gap: 12px; align-items: center; margin-bottom: 8px; min-width: 0; }
  .log-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; flex: 1; }
  .method { font-weight: bold; padding: 2px 8px; border-radius: 4px; font-size: 13px; }
  .method-GET { background: #e8f5e9; color: #2e7d32; }
  .method-POST { background: #e3f2fd; color: #1565c0; }
  .status { font-weight: bold; }
  .status-2xx { color: #2e7d32; }
  .status-3xx { color: #f57f17; }
  .status-4xx { color: #d32f2f; }
  .status-5xx { color: #b71c1c; }
  .timestamp { color: #888; font-size: 13px; }
  .section { margin-top: 8px; }
  .section-title { font-size: 12px; font-weight: bold; color: #666; text-transform: uppercase; margin-bottom: 4px; }
  pre { background: #f5f5f5; padding: 8px; border-radius: 4px; font-size: 13px; overflow-x: auto; margin: 0; white-space: pre-wrap; word-break: break-all; }
  .empty { color: #888; font-style: italic; text-align: center; padding: 40px; }
</style>
</head>
<body>
<nav>
  <a href="/ui/users">Users</a>
  <a href="/ui/groups">Groups</a>
  <a href="/ui/logs" class="active">Logs</a>
</nav>
<div class="container">
  <h1>OIDC Request Logs</h1>
  <div class="actions">
    <button class="btn btn-clear" onclick="clearLogs()">Clear Logs</button>
  </div>
  <div id="logs"></div>
</div>
<script>
function load() {
  fetch('/api/logs').then(r => r.json()).then(logs => {
    const el = document.getElementById('logs');
    if (!logs || logs.length === 0) {
      el.innerHTML = '<div class="empty">No OIDC requests logged yet.</div>';
      return;
    }
    el.innerHTML = logs.map(e => {
      const sc = e.ResponseStatus >= 500 ? '5xx' : e.ResponseStatus >= 400 ? '4xx' : e.ResponseStatus >= 300 ? '3xx' : '2xx';
      return '<div class="log-entry">' +
        '<div class="log-header">' +
          '<span class="method method-' + esc(e.Method) + '">' + esc(e.Method) + '</span>' +
          '<span class="log-path" title="' + esc(e.Path) + (e.Query ? '?' + esc(e.Query) : '') + '">' + esc(e.Path) + (e.Query ? '?' + esc(e.Query) : '') + '</span>' +
          '<span class="status status-' + sc + '">' + e.ResponseStatus + '</span>' +
          '<span class="timestamp">' + esc(e.Timestamp) + '</span>' +
        '</div>' +
        (e.RequestHeaders ? '<div class="section"><div class="section-title">Request Headers</div><pre>' + esc(e.RequestHeaders) + '</pre></div>' : '') +
        (e.RequestBody ? '<div class="section"><div class="section-title">Request Body</div><pre>' + esc(e.RequestBody) + '</pre></div>' : '') +
        (e.ResponseHeaders ? '<div class="section"><div class="section-title">Response Headers</div><pre>' + esc(e.ResponseHeaders) + '</pre></div>' : '') +
        (e.ResponseBody ? '<div class="section"><div class="section-title">Response Body</div><pre>' + esc(prettyJSON(e.ResponseBody)) + '</pre></div>' : '') +
      '</div>';
    }).join('');
  });
}
function esc(s) { if (!s) return ''; const d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
function prettyJSON(s) { try { return JSON.stringify(JSON.parse(s), null, 2); } catch(e) { return s; } }
function clearLogs() {
  if (!confirm('Clear all logs?')) return;
  fetch('/api/logs', { method: 'DELETE' }).then(() => load());
}
load();
setInterval(load, 3000);
</script>
</body>
</html>`))

var groupsTmpl = template.Must(template.New("groups").Parse(`<!DOCTYPE html>
<html>
<head>
<title>Groups - oauth-tester</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
  nav { background: #1a1a2e; padding: 12px 24px; display: flex; gap: 16px; }
  nav a { color: #e0e0e0; text-decoration: none; font-size: 14px; }
  nav a:hover { color: #fff; }
  nav a.active { color: #fff; font-weight: bold; }
  .container { max-width: 900px; margin: 24px auto; padding: 0 24px; }
  h1 { font-size: 24px; }
  .actions { display: flex; gap: 8px; margin-bottom: 16px; }
  .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
  .btn-add { background: #1a1a2e; color: #fff; }
  .btn-add:hover { background: #16213e; }
  .btn-del { background: #d32f2f; color: #fff; font-size: 12px; padding: 4px 10px; }
  .btn-del:hover { background: #b71c1c; }
  .btn-sm { font-size: 12px; padding: 4px 10px; background: #1a1a2e; color: #fff; }
  .btn-sm:hover { background: #16213e; }
  .group-card { background: #fff; border-radius: 8px; padding: 16px; margin-bottom: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
  .group-header { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
  .group-name { font-weight: bold; font-size: 18px; }
  .member-count { color: #888; font-size: 13px; }
  .members { display: flex; flex-wrap: wrap; gap: 6px; margin-top: 8px; }
  .member-tag { display: inline-flex; align-items: center; gap: 4px; background: #e8f5e9; color: #2e7d32; padding: 4px 10px; border-radius: 16px; font-size: 13px; }
  .member-tag .remove { cursor: pointer; font-weight: bold; margin-left: 2px; }
  .member-tag .remove:hover { color: #d32f2f; }
  .empty { color: #888; font-style: italic; }
</style>
</head>
<body>
<nav>
  <a href="/ui/users">Users</a>
  <a href="/ui/groups" class="active">Groups</a>
  <a href="/ui/logs">Logs</a>
</nav>
<div class="container">
  <h1>Groups</h1>
  <div class="actions">
    <button class="btn btn-add" onclick="addGroup()">Add Group</button>
  </div>
  <div id="groups"></div>
</div>
<script>
var allUsers = [];
function load() {
  Promise.all([
    fetch('/api/groups').then(r => r.json()),
    fetch('/api/users').then(r => r.json())
  ]).then(([groups, users]) => {
    allUsers = users;
    const el = document.getElementById('groups');
    if (!groups || groups.length === 0) {
      el.innerHTML = '<div class="empty">No groups yet. Click "Add Group" to create one.</div>';
      return;
    }
    Promise.all(groups.map(g =>
      fetch('/api/groups/' + encodeURIComponent(g.Name) + '/members').then(r => r.json()).then(members => ({group: g, members: members}))
    )).then(data => {
      el.innerHTML = data.map(({group, members}) => {
        const memberTags = members.map(m =>
          '<span class="member-tag">' + esc(m) + ' <span class="remove" onclick="removeMember(\'' + esc(group.Name) + '\', \'' + esc(m) + '\')">&times;</span></span>'
        ).join('');
        const nonMembers = allUsers.filter(u => !members.includes(u.Username));
        const addBtn = nonMembers.length > 0
          ? ' <button class="btn btn-sm" onclick="addMember(\'' + esc(group.Name) + '\')">Add Member</button>'
          : '';
        return '<div class="group-card">' +
          '<div class="group-header">' +
            '<span class="group-name">' + esc(group.Name) + '</span>' +
            '<span class="member-count">' + members.length + ' member' + (members.length !== 1 ? 's' : '') + '</span>' +
            addBtn +
            ' <button class="btn btn-del" onclick="delGroup(\'' + esc(group.Name) + '\')">Delete</button>' +
          '</div>' +
          '<div class="members">' + (memberTags || '<span class="empty">No members</span>') + '</div>' +
        '</div>';
      }).join('');
    });
  });
}
function esc(s) { if (!s) return ''; const d = document.createElement('div'); d.textContent = s; return d.innerHTML.replace(/'/g, "&#39;").replace(/"/g, '&quot;'); }
function addGroup() {
  const name = prompt('Group name:');
  if (!name) return;
  fetch('/api/groups', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({Name: name}) })
    .then(r => { if (!r.ok) return r.json().then(e => { alert(e.error); throw e; }); return r; })
    .then(() => load());
}
function delGroup(name) {
  if (!confirm('Delete group "' + name + '"? Members will be removed from this group.')) return;
  fetch('/api/groups/' + encodeURIComponent(name), { method: 'DELETE' }).then(() => load());
}
function addMember(groupName) {
  fetch('/api/groups/' + encodeURIComponent(groupName) + '/members').then(r => r.json()).then(members => {
    const nonMembers = allUsers.filter(u => !members.includes(u.Username));
    if (nonMembers.length === 0) { alert('All users are already in this group.'); return; }
    const choice = prompt('Add user to "' + groupName + '":\\n\\n' + nonMembers.map((u, i) => (i+1) + '. ' + u.Username).join('\\n') + '\\n\\nEnter number or username:');
    if (!choice) return;
    const num = parseInt(choice);
    const username = (num > 0 && num <= nonMembers.length) ? nonMembers[num-1].Username : choice;
    fetch('/api/groups/' + encodeURIComponent(groupName) + '/members', {
      method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({Username: username})
    }).then(r => { if (!r.ok) return r.json().then(e => { alert(e.error); throw e; }); return r; })
      .then(() => load());
  });
}
function removeMember(groupName, username) {
  fetch('/api/groups/' + encodeURIComponent(groupName) + '/members/' + encodeURIComponent(username), { method: 'DELETE' }).then(() => load());
}
load();
</script>
</body>
</html>`))

func (p *PageHandlers) UsersPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	usersTmpl.Execute(w, nil)
}

func (p *PageHandlers) GroupsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	groupsTmpl.Execute(w, nil)
}

func (p *PageHandlers) LogsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	logsTmpl.Execute(w, nil)
}

// ListLogs returns logs as JSON. Supports query parameters for filtering:
//
//	method   - HTTP method (e.g. POST)
//	path     - request path (e.g. /token)
//	status   - response status code (e.g. 400)
//	limit    - max results (default 50, used by UI with 200)
//	minutes  - only entries from the last N minutes
//
// When no query parameters are provided, behaves like the original endpoint
// (returns up to 200 recent entries) for backward compatibility with the UI.
func (a *APIHandlers) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hasFilters := q.Has("method") || q.Has("path") || q.Has("status") || q.Has("limit") || q.Has("minutes")

	var logs []store.LogEntry
	var err error

	if hasFilters {
		lq := store.LogQuery{}
		lq.Method = strings.ToUpper(q.Get("method"))
		lq.Path = q.Get("path")
		if v := q.Get("status"); v != "" {
			lq.Status, _ = strconv.Atoi(v)
		}
		if v := q.Get("limit"); v != "" {
			lq.Limit, _ = strconv.Atoi(v)
		}
		if v := q.Get("minutes"); v != "" {
			lq.MinutesAgo, _ = strconv.Atoi(v)
		}
		logs, err = a.Store.QueryLogs(lq)
	} else {
		logs, err = a.Store.ListLogs(200)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []store.LogEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// ClearLogs deletes all log entries.
func (a *APIHandlers) ClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := a.Store.ClearLogs(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
