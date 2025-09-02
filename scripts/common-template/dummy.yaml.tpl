openapi: 3.0.3
{{- $data := (ds "data") }}
{{- $path := (ds "path") }}
{{- $fullPath := printf "%s/%s" $path.dir $path.file }}

paths:
{{- if coll.Has $data "schemas" }}
  {{- range $schemaName, $schema := $data.schemas }}
  /dummy/{{ $schemaName | strings.Slug }}:
    get:
      summary: Get dummy {{ $schemaName }}
      description: |
        Returns a dummy response for the {{ $schemaName }} schema.
      operationId: getDummy{{ $schemaName | strings.Title | strings.ReplaceAll " " "" }}
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{ $schemaName }}Excluded'
  {{- end }}
{{- end }}

{{- if coll.Has $data "parameters" }}
  {{- range $paramName, $param := $data.parameters }}
  /dummy/parameters/{{ $name := $paramName | strings.Slug }}{{ $name }}{{ if (eq $param.in "path") }}{{ if (eq $param.name "name") }}{{ print "/{name}" }}{{ else }}{{ printf "/{%s}" $paramName }}{{ end }}{{ end }}:
    get:
      summary: Test parameter {{ $paramName }}
      description: |
        Dummy endpoint for testing the {{ $paramName }} parameter.
      operationId: testParameter{{ $paramName | strings.Title | strings.ReplaceAll " " "" }}
      parameters:
        - $ref: '#/components/parameters/{{ $paramName }}'
      responses:
        '200':
          description: Parameter test successful
  {{- end }}
{{- end }}

{{- if and (not (coll.Has $data "schemas")) (not (coll.Has $data "parameters")) }}
  {{- range $itemName, $item := $data }}
  /dummy/{{ $itemName | strings.Slug }}:
    get:
      summary: Get dummy {{ $itemName }}
      description: |
        Returns a dummy response for the {{ $itemName }} item.
      operationId: getDummy{{ $itemName | strings.Title | strings.ReplaceAll " " "" }}
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/{{ $itemName }}Excluded'
  {{- end }}
{{- end }}

components:
{{- if coll.Has $data "schemas" }}
  schemas:
{{- range $schemaName, $schema := $data.schemas }}
    {{ $schemaName }}Excluded:
        description: Dummy for {{ $schemaName }}
        type: object
        required:
            - items
        properties:
            items:
                description: Dummy of {{ $schemaName }}
                type: array
                items:
                    $ref: '{{ printf "%s#/schemas/%s" $fullPath $schemaName }}'
{{- end }}
{{- else if and (not (coll.Has $data "parameters")) (not (coll.Has $data "schemas")) }}
  schemas:
{{- range $itemName, $item := $data }}
    {{ $itemName }}Excluded:
        description: Dummy for {{ $itemName }}
        type: object
        required:
            - items
        properties:
            items:
                description: Dummy of {{ $itemName }}
                type: array
                items:
                    $ref: '{{ printf "%s#/%s" $fullPath $itemName }}'
{{- end }}
{{- end }}

{{- if coll.Has $data "parameters" }}
  parameters:
{{- range $paramName, $param := $data.parameters }}
    {{ $paramName }}:
{{ $param | data.ToYAML | strings.ReplaceAll "'#/" "'#/components/parameters/" | indent 6 }}
{{- end }}
{{- end }}
