build_windows:
	GOOS=windows GOARCH=amd64 go build

package:
	rm results/*
	rmdir results
	cd ..
	rm sim.zip
	zip -r sim.zip sim/ -x *.git*