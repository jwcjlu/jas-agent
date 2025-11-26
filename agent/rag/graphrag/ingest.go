package graphrag

import (
	"container/list"
	"jas-agent/agent/rag/loader"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func (e *Engine) ingestDocument(doc *loader.Document) (addedNodes int, addedEdges int) {
	e.store.addDocument(doc)
	sentences := splitSentences(doc.Text)
	seenNodes := map[string]struct{}{}

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		entities := extractEntities(sentence)
		if len(entities) == 0 {
			continue
		}
		for _, entity := range entities {
			nodeID := normalizeEntity(entity)
			if nodeID == "" {
				continue
			}
			e.store.upsertNode(nodeID, func(node *loader.GraphNode, created bool) {
				if created {
					addedNodes++
					node.Name = entity
				} else if len(entity) > len(node.Name) && strings.Count(entity, " ") <= 4 {
					node.Name = entity
				}
				if node.Metadata == nil {
					node.Metadata = map[string]string{}
				}
				for k, v := range doc.Metadata {
					if _, exists := node.Metadata[k]; !exists {
						node.Metadata[k] = v
					}
				}
				if node.SourceDocs == nil {
					node.SourceDocs = map[string]int{}
				}
				node.SourceDocs[doc.ID]++
				node.Occurrence++
				if node.Summary == "" {
					node.Summary = sentence
				} else {
					node.Summary = truncateSummary(node.Summary, sentence, e.options.MaxSummaryLength)
				}
				if len(node.Snippets) < e.options.MaxSnippetsPerNode {
					node.Snippets = appendUniqueSnippet(node.Snippets, sentence)
				}
			})
			seenNodes[nodeID] = struct{}{}
		}

		// 关系
		for i := 0; i < len(entities); i++ {
			for j := i + 1; j < len(entities); j++ {
				source := normalizeEntity(entities[i])
				target := normalizeEntity(entities[j])
				if source == "" || target == "" || source == target {
					continue
				}
				addedEdges++
				e.store.addEdge(&loader.GraphEdge{
					Source:   source,
					Target:   target,
					Relation: inferRelation(sentence),
					Evidence: sentence,
					Weight:   computeEdgeWeight(sentence),
				})
			}
		}
	}
	return addedNodes, addedEdges
}

func splitSentences(text string) []string {
	replacer := strings.NewReplacer("\n", "。", "?", "。", "!", "。", ";", "。")
	normalized := replacer.Replace(text)
	parts := strings.Split(normalized, "。")
	var sentences []string
	for _, sentence := range parts {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}
	return sentences
}

func extractEntities(sentence string) []string {
	var entities []string
	var current strings.Builder
	lastHan := false
	addEntity := func() {
		if current.Len() == 0 {
			return
		}
		entity := strings.TrimSpace(current.String())
		current.Reset()
		if len([]rune(entity)) < 2 {
			return
		}
		normalized := normalizeEntity(entity)
		if normalized == "" {
			return
		}
		entities = append(entities, entity)
	}

	for _, r := range sentence {
		switch {
		case unicode.Is(unicode.Han, r):
			if !lastHan {
				addEntity()
			}
			current.WriteRune(r)
			lastHan = true
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if lastHan {
				addEntity()
			}
			current.WriteRune(r)
			lastHan = false
		default:
			addEntity()
			lastHan = false
		}
	}
	addEntity()
	return deduplicateEntities(entities)
}

func deduplicateEntities(entities []string) []string {
	seen := make(map[string]struct{}, len(entities))
	var result []string
	for _, entity := range entities {
		key := normalizeEntity(entity)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, entity)
	}
	return result
}

func normalizeEntity(entity string) string {
	entity = strings.TrimSpace(entity)
	entity = strings.Trim(entity, "-_#.,:;!?()[]{}\"'")
	entity = strings.ReplaceAll(entity, "\t", " ")
	entity = strings.ReplaceAll(entity, "\n", " ")
	entity = strings.Join(strings.Fields(entity), " ")
	entity = strings.ToLower(entity)
	if len([]rune(entity)) < 2 {
		return ""
	}
	return entity
}

func inferRelation(sentence string) string {
	verbs := []string{
		"依赖", "使用", "连接", "负责", "包含", "组成", "影响", "导致", "支持",
		"depends on", "uses", "connects to", "responsible for", "contains", "affects",
	}
	lower := strings.ToLower(sentence)
	for _, verb := range verbs {
		if strings.Contains(lower, verb) {
			return verb
		}
	}
	return "related_to"
}

func computeEdgeWeight(sentence string) float64 {
	base := 1.0
	length := float64(len([]rune(sentence)))
	if length > 120 {
		return base * 0.8
	}
	if length < 40 {
		return base * 1.2
	}
	return base
}

func appendUniqueSnippet(snippets []string, snippet string) []string {
	for _, s := range snippets {
		if s == snippet {
			return snippets
		}
	}
	return append(snippets, snippet)
}

func truncateSummary(base, addition string, limit int) string {
	if addition == "" {
		return base
	}
	if base == "" {
		if len([]rune(addition)) > limit {
			return string([]rune(addition)[:limit])
		}
		return addition
	}
	combined := base + " " + addition
	runes := []rune(combined)
	if len(runes) <= limit {
		return combined
	}
	return string(runes[:limit])
}

func (e *Engine) rebuildCommunities() {
	nodes := e.store.listNodes()
	if len(nodes) == 0 {
		e.store.updateCommunities(map[string]*loader.Community{})
		return
	}
	adj := make(map[string][]string)
	for _, edge := range e.store.listEdges() {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		adj[edge.Target] = append(adj[edge.Target], edge.Source)
	}

	communities := map[string]*loader.Community{}
	visited := map[string]bool{}
	idx := 0
	for _, node := range nodes {
		if visited[node.ID] {
			continue
		}
		idx++
		queue := list.New()
		queue.PushBack(node.ID)
		var memberIDs []string
		var snippets []string
		for queue.Len() > 0 {
			element := queue.Front()
			queue.Remove(element)
			currentID := element.Value.(string)
			if visited[currentID] {
				continue
			}
			visited[currentID] = true
			memberIDs = append(memberIDs, currentID)
			if node, ok := e.store.getNode(currentID); ok {
				if node.Summary != "" {
					snippets = append(snippets, node.Summary)
				}
				snippets = append(snippets, node.Snippets...)
			}
			for _, neighbor := range adj[currentID] {
				if !visited[neighbor] {
					queue.PushBack(neighbor)
				}
			}
		}
		communityID := "community-" + strconv.Itoa(idx)
		communities[communityID] = &loader.Community{
			ID:        communityID,
			NodeIDs:   memberIDs,
			Summary:   buildCommunitySummary(memberIDs, snippets),
			Keywords:  collectKeywords(snippets),
			UpdatedAt: time.Now(),
		}
	}
	e.store.updateCommunities(communities)
}

func buildCommunitySummary(nodeIDs []string, snippets []string) string {
	var builder strings.Builder
	builder.WriteString("Community contains entities: ")
	builder.WriteString(strings.Join(nodeIDs, ", "))
	if len(snippets) > 0 {
		builder.WriteString(". Key facts: ")
		limit := 3
		if len(snippets) < limit {
			limit = len(snippets)
		}
		for i := 0; i < limit; i++ {
			builder.WriteString(snippets[i])
			if i != limit-1 {
				builder.WriteString("; ")
			}
		}
	}
	return builder.String()
}

func collectKeywords(snippets []string) []string {
	counter := map[string]int{}
	for _, snippet := range snippets {
		for _, token := range tokenize(snippet) {
			if len(token) <= 2 {
				continue
			}
			counter[token]++
		}
	}
	type kv struct {
		key   string
		count int
	}
	var pairs []kv
	for k, v := range counter {
		pairs = append(pairs, kv{key: k, count: v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].key < pairs[j].key
		}
		return pairs[i].count > pairs[j].count
	})
	limit := 6
	if len(pairs) < limit {
		limit = len(pairs)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, pairs[i].key)
	}
	return result
}
