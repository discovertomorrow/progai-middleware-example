[progai-middleware](https://github.com/discovertomorrow/progai-middleware) is
an experimental middleware to queue and serve ai endpoints.

The commands in `./cmd` are examples of how to use the package.

While developed by prognosticians, progai-middleware is not an official
prognostica product.

```bash
go run ./cmd/llamacpp \
  --endpoint=http://10.0.0.1:8090/completion \
  --slots=1 \
  --stop "</s>, [INST], [/INST]" \
  --template '{{ range . }}
{{- if eq .Role "system" }} [INST] <system> {{ .Content }} </system> [/INST]
{{- else if eq .Role "user" }} [INST] {{ .Content }} [/INST]
{{- else if eq .Role "tool" }} [INST] <toolresult> {{ .Content }} </toolresult> [/INST]
{{- else }} {{ .Content }}
{{- if .ToolCalls }}{{ range .ToolCalls }}<toolcall> {{ .Function.Name }} with arguments {{ .Function.Arguments }} </toolcall> {{ end }}{{ end -}}
</s>{{ end }}
{{- end }}'
```
