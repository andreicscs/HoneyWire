import { ref, nextTick, type Ref } from 'vue'

// Shared inline-editing state for node fields (alias, IPs, tags).
// Used by both NodeDetailHeader and FleetNodeWidget so the edit flows stay
// identical: enable -> edit -> Enter/blur saves (emits `update`), Esc cancels.
//
// The input element refs stay in the components (template `ref` attributes
// only link to directly-declared setup bindings) and are passed in here.
export interface NodeFieldInputs {
  aliasInput: Ref<HTMLInputElement | null>
  pubIpInput: Ref<HTMLInputElement | null>
  privIpInput: Ref<HTMLInputElement | null>
  tagInput: Ref<HTMLInputElement | null>
}

export function useNodeFieldEdit(
  getNode: () => any,
  emit: (e: 'update', updates: any) => void,
  inputs: NodeFieldInputs
) {
  const { aliasInput, pubIpInput, privIpInput, tagInput } = inputs

  // --- Alias ---
  const isEditingAlias = ref(false)
  const aliasValue = ref('')
  const enableAliasEdit = async () => {
    const node = getNode()
    if (!node) return
    isEditingAlias.value = true
    aliasValue.value = node.alias
    await nextTick()
    aliasInput.value?.focus()
    aliasInput.value?.select()
  }
  const cancelAliasEdit = () => {
    isEditingAlias.value = false
    aliasValue.value = ''
  }
  const saveAlias = () => {
    const node = getNode()
    if (!isEditingAlias.value || !node) return
    const val = aliasValue.value.trim()
    if (val && val !== node.alias) emit('update', { alias: val })
    isEditingAlias.value = false
  }

  // --- Public IP ---
  const isEditingPubIp = ref(false)
  const pubIpValue = ref('')
  const enablePubIpEdit = async () => {
    const node = getNode()
    if (!node) return
    isEditingPubIp.value = true
    pubIpValue.value = node.publicIp || ''
    await nextTick()
    pubIpInput.value?.focus()
    pubIpInput.value?.select()
  }
  const cancelPubIpEdit = () => {
    isEditingPubIp.value = false
    pubIpValue.value = ''
  }
  const savePubIp = () => {
    const node = getNode()
    if (!isEditingPubIp.value || !node) return
    const val = pubIpValue.value.trim()
    if (val !== (node.publicIp || '')) emit('update', { publicIp: val })
    isEditingPubIp.value = false
  }

  // --- Private IP ---
  const isEditingPrivIp = ref(false)
  const privIpValue = ref('')
  const enablePrivIpEdit = async () => {
    const node = getNode()
    if (!node) return
    isEditingPrivIp.value = true
    privIpValue.value = node.privateIp || ''
    await nextTick()
    privIpInput.value?.focus()
    privIpInput.value?.select()
  }
  const cancelPrivIpEdit = () => {
    isEditingPrivIp.value = false
    privIpValue.value = ''
  }
  const savePrivIp = () => {
    const node = getNode()
    if (!isEditingPrivIp.value || !node) return
    const val = privIpValue.value.trim()
    if (val !== (node.privateIp || '')) emit('update', { privateIp: val })
    isEditingPrivIp.value = false
  }

  // --- Tags ---
  const isEditingTag = ref(false)
  const newTagValue = ref('')
  const enableTagEdit = async () => {
    isEditingTag.value = true
    await nextTick()
    tagInput.value?.focus()
  }
  const cancelTag = () => {
    isEditingTag.value = false
    newTagValue.value = ''
  }
  const saveTag = () => {
    const node = getNode()
    if (!isEditingTag.value || !node) return
    const val = newTagValue.value.trim()
    if (val && !node.tags.includes(val)) emit('update', { tags: [...node.tags, val] })
    isEditingTag.value = false
    newTagValue.value = ''
  }
  const removeTag = (index: number | string) => {
    const node = getNode()
    if (!node) return
    const newTags = [...node.tags]
    newTags.splice(Number(index), 1)
    emit('update', { tags: newTags })
  }

  return {
    isEditingAlias, aliasValue, enableAliasEdit, cancelAliasEdit, saveAlias,
    isEditingPubIp, pubIpValue, enablePubIpEdit, cancelPubIpEdit, savePubIp,
    isEditingPrivIp, privIpValue, enablePrivIpEdit, cancelPrivIpEdit, savePrivIp,
    isEditingTag, newTagValue, enableTagEdit, cancelTag, saveTag, removeTag,
  }
}
