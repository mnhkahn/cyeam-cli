{{ range .Versions }}{{ range .CommitGroups }}{{ range .Commits }}- {{ .Subject }}
{{ end }}{{ end }}{{ end }}