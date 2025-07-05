package utils

// ChooseStorageNodes выбирает серверы хранения для чанка
// index - индекс чанка в файле
// storagePool - список доступных серверов хранения
// replicaCount - количество реплик (по умолчанию 2)
func ChooseStorageNodes(index int, storagePool []string, replicaCount int) []string {
	if replicaCount <= 0 {
		replicaCount = 2
	}

	if len(storagePool) <= replicaCount {
		return storagePool
	}

	result := make([]string, 0, replicaCount)
	poolSize := len(storagePool)

	// Начинаем с узла, соответствующего индексу чанка
	primary := index % poolSize
	result = append(result, storagePool[primary])

	// Добавляем дополнительные узлы для реплик
	for i := 1; i < replicaCount; i++ {
		nextNode := (primary + i) % poolSize
		result = append(result, storagePool[nextNode])
	}

	return result
}
