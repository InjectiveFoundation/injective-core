all:

testupgrade-v114:
	go test -race -v -run TestUpgradeV114

coverupgrade-v114:
	go test -race -cover -v -coverprofile=upgrade-coverage.out -run TestUpgradeV114
	go tool cover -html=upgrade-coverage.out -o upgrade-coverage.html
	open upgrade-coverage.html

testupgrade: testupgrade-v114

coverupgrade: coverupgrade-v114

tu: testupgrade

cu: coverupgrade

cuclean:
	rm upgrade-coverage.html upgrade-coverage.out

.PHONY: cu cuclean coverupgrade coverupgrade-v114 tu testupgrade testupgrade-v114

