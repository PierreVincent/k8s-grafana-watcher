package main

import (
	"fmt"
	"log"
	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kselector "k8s.io/kubernetes/pkg/fields"
	klabels "k8s.io/kubernetes/pkg/labels"
	"crypto/sha1"
)

type ConfigMapLookup struct {
	Annotation string
	hashes     map[string]string
}

func NewConfigMapLookup(annotation string) *ConfigMapLookup {
	return &ConfigMapLookup{
		Annotation: annotation,
		hashes:     make(map[string]string),
	}
}

type ConfigMapEntry struct {
	Namespace string
	Name      string
	Key       string
	Value     string
}

func (entry *ConfigMapEntry) identifier() string {
	return fmt.Sprintf("%s-%s-%s", entry.Namespace, entry.Name, entry.Key)
}

func (lookup *ConfigMapLookup) FindNewEntries(kubeClient *kclient.Client) []ConfigMapEntry {
	allEntries := lookup.gatherEntries(kubeClient)
	newEntries := []ConfigMapEntry{}
	for _, entry := range allEntries {
		hash := computeSha1(entry.Value)
		entryIdentifier := entry.identifier()
		if lookup.hashes[entryIdentifier] != hash {
			newEntries = append(newEntries, entry)
			lookup.hashes[entryIdentifier] = hash
		}
	}
	return newEntries
}

func (lookup *ConfigMapLookup) gatherEntries(kubeClient *kclient.Client) []ConfigMapEntry {
	si := kubeClient.ConfigMaps(kapi.NamespaceAll)
	mapList, err := si.List(kapi.ListOptions{
		LabelSelector: klabels.Everything(),
		FieldSelector: kselector.Everything()})
	if err != nil {
		log.Printf("Unable to list configmaps: %s", err)
	}

	entryList := []ConfigMapEntry{}

	for _, cm := range mapList.Items {
		anno := cm.GetObjectMeta().GetAnnotations()
		name := cm.GetObjectMeta().GetName()
		namespace := cm.GetObjectMeta().GetNamespace()

		for k := range anno {
			if k == lookup.Annotation {
				for cmk, cmv := range cm.Data {
					entry := ConfigMapEntry{namespace, name, cmk, cmv }
					entryList = append(entryList, entry)
				}
			}
		}
	}
	return entryList
}

func computeSha1(payload string) string {
	hash := sha1.New()
	hash.Write([]byte(payload))

	return fmt.Sprintf("%x", hash.Sum(nil))
}
