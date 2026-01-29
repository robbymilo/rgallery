curl -s "http://localhost:3000/api/timeline" | jq > ./testdata/ResponseFilter.json
curl -s "http://localhost:3000/api/timeline?orderby=modified" | jq > ./testdata/ResponseFilter-modified.json
curl -s "http://localhost:3000/api/timeline?orderby=modified&direction=asc" | jq > ./testdata/ResponseFilter-modified-asc.json
curl -s "http://localhost:3000/api/timeline?term=copp" | jq > ./testdata/ResponseFilter-term.json
curl -s "http://localhost:3000/api/timeline?term=su+špa" | jq > ./testdata/ResponseFilter-term-1.json
curl -s "http://localhost:3000/api/timeline?camera=NIKON%20D800" | jq > ./testdata/ResponseFilter-camera.json
curl -s "http://localhost:3000/api/timeline?lens=AF-S%20Nikkor%2050mm%20f%2f1.8G" | jq > ./testdata/ResponseFilter-lens.json
curl -s "http://localhost:3000/api/timeline?lens=123" | jq > ./testdata/ResponseFilter-lens-1.json
curl -s "http://localhost:3000/api/timeline?lens=Nikon%20Ai-s%20105mm%20f%2f2.5" | jq > ./testdata/ResponseFilter-lens-2.json
curl -s "http://localhost:3000/api/timeline?folder=2017/20170624-idaho" | jq > ./testdata/ResponseFilter-folder.json
curl -s "http://localhost:3000/api/timeline?tag=idaho" | jq > ./testdata/ResponseFilter-tag.json
curl -s "http://localhost:3000/api/memories" | jq > ./testdata/ResponseFilter-memories.json

curl -s "http://localhost:3000/api/media/651935749" | jq > ./testdata/ResponseImage-0.json
curl -s "http://localhost:3000/api/media/3455659031" | jq > ./testdata/ResponseImage-1.json
curl -s "http://localhost:3000/api/media/4119775194" | jq > ./testdata/ResponseImage-2.json
curl -s "http://localhost:3000/api/media/651935749?folder=2017/20170624-idaho" | jq > ./testdata/ResponseImage-folder.json
curl -s "http://localhost:3000/api/media/651935749?tag=idaho" | jq > ./testdata/ResponseImage-tag.json
curl -s "http://localhost:3000/api/media/4119775194?tag=%40acconfb" | jq > ./testdata/ResponseImage-tag-acc.json
curl -s "http://localhost:3000/api/media/4119775194?tag=%23californiawildfires" | jq > ./testdata/ResponseImage-tag-cal.json
curl -s "http://localhost:3000/api/media/651935749?rating=5" | jq > ./testdata/ResponseImage-favorites.json

# prev/next responses
curl -s "http://localhost:3000/api/media/3455659031?camera=NIKON%20D800&format=json" | jq > ./testdata/ResponseImage-camera.json
curl -s "http://localhost:3000/api/media/651935749?lens=AF-S%20Nikkor%2050mm%20f%2f1.8G&format=json" | jq > ./testdata/ResponseImage-lens.json
curl -s "http://localhost:3000/api/media/525791494?lens=Nikon%20Ai-s%20105mm%20f%2f2.5&format=json" | jq > ./testdata/ResponseImage-lens-1.json
curl -s "http://localhost:3000/api/media/264898052?focallength35=50&format=json" | jq > ./testdata/ResponseImage-focallength35.json
curl -s "http://localhost:3000/api/media/3216513272?software=darktable%204.4.2&format=json" | jq > ./testdata/ResponseImage-software.json
curl -s "http://localhost:3000/api/media/1129346697?term=bogus&format=json" | jq > ./testdata/ResponseImage-term.json

curl -s "http://localhost:3000/api/folders?format=json" | jq > ./testdata/ResponseFolders.json


curl -s "http://localhost:3000/api/tags" | jq > ./testdata/ResponseTags.json

curl -s "http://localhost:3000/api/gear" | jq > ./testdata/ResponseGear.json

curl -s "http://localhost:3000/api/map" | jq > ./testdata/ResponseMap.json
