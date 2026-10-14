import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import NodeCatalogGrid from '../NodeCatalogGrid.vue'

const manifestsFixture = () => [
  { id: 'hw-a', name: 'Sensor A', description: 'Watches A.', osi_layer: 'Network', icon_svg: 'M1' },
  { id: 'hw-b', name: 'Sensor B', description: 'Watches B.', osi_layer: 'Application', icon_svg: 'M2' },
]

const installedFixture = () => [{ id: 'hw-a', sensorId: 'hw-a', name: 'hw-a' }]

function mountCatalog(overrides = {}) {
  return mount(NodeCatalogGrid, {
    props: { manifests: [], isLoading: false, fetchError: false, installedSensors: [], ...overrides },
    attachTo: document.body,
  })
}

describe('NodeCatalogGrid', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders the catalog title', () => {
    const wrapper = mountCatalog()
    expect(wrapper.text()).toContain('Sensor Catalog')
  })

  it('shows the fetch error state', () => {
    const wrapper = mountCatalog({ fetchError: true })
    expect(wrapper.text()).toContain('Unable to reach Sensor Registry')
  })

  it('shows the empty state without manifests', () => {
    const wrapper = mountCatalog()
    expect(wrapper.text()).toContain('No Sensors Available')
  })

  it('renders one card per manifest', () => {
    const wrapper = mountCatalog({ manifests: manifestsFixture(), installedSensors: installedFixture() })
    const cards = wrapper.findAll('[data-testid="catalog-card"]')
    expect(cards).toHaveLength(2)
    expect(wrapper.text()).toContain('Sensor A')
    expect(wrapper.text()).toContain('Sensor B')
  })

  it('emits open for uninstalled sensors', async () => {
    const manifests = manifestsFixture()
    const wrapper = mountCatalog({ manifests, installedSensors: installedFixture() })
    await wrapper.find('[data-manifest-id="hw-b"]').trigger('click')
    expect(wrapper.emitted('open')?.[0]).toEqual([manifests[1]])
  })

  it('emits edit for installed sensors', async () => {
    const installed = installedFixture()
    const wrapper = mountCatalog({ manifests: manifestsFixture(), installedSensors: installed })
    await wrapper.find('[data-manifest-id="hw-a"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual([installed[0]])
  })
})
