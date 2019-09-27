package main

const bookmarkTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>bm</title>
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap.min.css">
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/css/bootstrap-theme.min.css">
<script src="https://ajax.googleapis.com/ajax/libs/jquery/1.11.3/jquery.min.js"></script>
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.5/js/bootstrap.min.js"></script>
<style type="text/css">
    .bookmark{
    	margin: 20px;
	}
	.customImage {
		width: 50px;
		height: 50px;
	}
</style>
</head>
<body>
<div class="bookmark">
    <table class="table table-hover">
        <thead>
            <tr>
				<th>URL</th>
				<th>Icon</th>
                <th>Last modified</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
			<tr>
					{{ $map := .BookmarkMap }}
					{{ with .Sorted }}
						{{range $key := .}}
							{{ with $map }}
								{{ $value := index $map $key }}
								<tr>
									<td> <a href={{$value.Url}}>{{ $value.Title }}</a></td>
									<td> <img class="customImage" src={{$value.Icon}}></td>
									<td> {{ humanize $value.Modified }}</td>
									<td> <a href='/remove/{{ $key }}'>remove</a></td>
								</tr>
								{{ end }}
							{{end}}
					{{end}}
			</tr>
        </tbody>
    </table>
</div>
</body>
</html>
`
