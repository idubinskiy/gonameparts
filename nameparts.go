package gonameparts

import (
	"sort"
	"strings"
)

type NameParts struct {
	ProvidedName string      `json:"provided_name"`
	FullName     string      `json:"full_name"`
	Salutation   string      `json:"salutation"`
	FirstName    string      `json:"first_name"`
	MiddleName   string      `json:"middle_name"`
	LastName     string      `json:"last_name"`
	Generation   string      `json:"generation"`
	Suffix       string      `json:"suffix"`
	Nickname     string      `json:"nickname"`
	Aliases      []NameParts `json:"aliases"`
}

type nameString struct {
	FullName  string
	SplitName []string
	Nickname  string
	Aliases   []string
}

func (n *nameString) cleaned() []string {
	unwanted := []string{",", "."}
	cleaned := []string{}
	for _, x := range n.split() {
		for _, y := range unwanted {
			x = strings.Replace(x, y, "", -1)
		}
		cleaned = append(cleaned, strings.Trim(x, " "))
	}
	return cleaned
}

func (n *nameString) searchParts(parts *[]string) int {
	for i, x := range n.cleaned() {
		for _, y := range *parts {
			if strings.ToUpper(x) == strings.ToUpper(y) {
				return i
			}
		}
	}
	return -1
}

func (n *nameString) looksCorporate() bool {
	return n.searchParts(&corpEntity) > -1
}

func (n *nameString) hasComma() bool {

	for _, x := range n.split() {
		if strings.ContainsAny(x, ",") {
			return true
		}
	}
	return false
}

func (n *nameString) slotNickname() {

	var nickNameBoundaries []int

	for index, x := range n.split() {
		if string(x[0]) == "'" || string(x[0]) == "\"" {
			nickNameBoundaries = append(nickNameBoundaries, index)
		}
		if string(x[len(x)-1]) == "'" || string(x[len(x)-1]) == "\"" {
			nickNameBoundaries = append(nickNameBoundaries, index)
		}
	}

	if len(nickNameBoundaries) > 0 && len(nickNameBoundaries)%2 == 0 {
		nickStart := nickNameBoundaries[0]
		nickEnd := nickNameBoundaries[1]

		nick := n.SplitName[:nickStart]
		postNick := n.SplitName[nickEnd+1:]

		n.Nickname = strings.Join(n.SplitName[nickStart:nickEnd+1], " ")
		nick = append(nick, postNick...)
		n.FullName = strings.Join(nick, " ")
	}
}

func (n *nameString) hasAliases() (bool, string) {
	for _, x := range nonName {
		if strings.Contains(strings.ToUpper(n.FullName), x) {
			return true, x
		}
	}
	return false, ""
}

func (n *nameString) find(part string) int {
	switch part {
	case "salutation":
		return n.searchParts(&salutations)
	case "generation":
		return n.searchParts(&generations)
	case "suffix":
		return n.searchParts(&suffixes)
	case "lnprefix":
		return n.searchParts(&lnPrefixes)
	case "nonname":
		return n.searchParts(&nonName)
	case "supplemental":
		return n.searchParts(&supplementalInfo)
	default:

	}
	return -1
}

func (n *nameString) split() []string {
	n.SplitName = strings.Fields(n.FullName)
	return n.SplitName
}

func (n *nameString) normalize() []string {
	hasAlias, aliasSep := n.hasAliases()

	if hasAlias {
		n.splitAliases(aliasSep)
	}

	n.slotNickname()

	if n.hasComma() {
		commaSplit := strings.SplitN(n.FullName, ",", 2)
		sort.StringSlice(commaSplit).Swap(1, 0)
		n.FullName = strings.Join(commaSplit, " ")
	}

	return n.cleaned()

}

func (n *nameString) splitAliases(aliasSep string) {
	splitNames := n.split()

	for index, part := range splitNames {
		if strings.ToUpper(part) == aliasSep {
			splitNames[index] = "*|*"
		}
	}

	names := strings.Split(strings.Join(splitNames, " "), "*|*")

	n.FullName = names[0]
	n.Aliases = names[1:]
}

func (p *NameParts) slot(part string, value string) {
	switch part {
	case "salutation":
		p.Salutation = value
	case "generation":
		p.Generation = value
	case "suffix":
		p.Suffix = value
	case "middle":
		p.MiddleName = value
	case "last":
		p.LastName = value
	case "first":
		p.FirstName = value
	default:

	}

}

func min(indexA int, indexB int) int {
	if indexA < indexB {
		return indexA
	}
	return indexB
}

func (n *nameString) findNotSlotted(slotted []int) []int {

	var notSlotted []int

	for i, _ := range n.SplitName {
		found := false
		for _, j := range slotted {
			if i == j {
				found = true
				break
			}
		}
		if !found {
			notSlotted = append(notSlotted, i)
		}
	}

	return notSlotted

}

func Parse(name string) NameParts {
	n := nameString{FullName: name}
	n.normalize()
	p := NameParts{ProvidedName: name, Nickname: n.Nickname}

	parts := []string{"salutation", "generation", "suffix", "lnprefix", "nonname", "supplemental"}
	partMap := make(map[string]int)
	var slotted []int

	// Slot Salutation, Generation and Suffix
	for _, part := range parts {
		partIndex := n.find(part)
		partMap[part] = partIndex
		if partIndex > -1 {
			p.slot(part, n.SplitName[partIndex])
			slotted = append(slotted, partIndex)
		}
	}

	// Slot FirstName
	partMap["first"] = partMap["salutation"] + 1
	p.slot("first", n.SplitName[partMap["first"]])
	slotted = append(slotted, partMap["salutation"]+1)

	// Slot prefixed LastName
	if partMap["lnprefix"] > -1 {
		lnEnd := len(n.SplitName)
		if partMap["generation"] > -1 {
			lnEnd = min(lnEnd, partMap["generation"])
		}
		if partMap["suffix"] > -1 {
			lnEnd = min(lnEnd, partMap["suffix"])
		}
		p.slot("last", strings.Join(n.SplitName[partMap["lnprefix"]:lnEnd], " "))

		// Keep track of what we've slotted
		for i := partMap["lnprefix"]; i <= lnEnd; i++ {
			slotted = append(slotted, i)
		}
	}

	// Slot the rest
	notSlotted := n.findNotSlotted(slotted)

	if len(notSlotted) == 2 {
		p.slot("middle", n.SplitName[notSlotted[0]])
		p.slot("last", n.SplitName[notSlotted[1]])
	}

	if len(notSlotted) == 1 {
		p.slot("last", n.SplitName[notSlotted[0]])
	}

	// Process aliases
	for _, alias := range n.Aliases {
		p.Aliases = append(p.Aliases, Parse(alias))
	}

	return p
}
