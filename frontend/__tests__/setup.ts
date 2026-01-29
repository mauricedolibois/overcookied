import '@testing-library/dom'

// Mock localStorage
const localStorageMock = {
  store: {} as Record<string, string>,
  getItem: function (key: string) {
    return this.store[key] || null
  },
  setItem: function (key: string, value: string) {
    this.store[key] = value
  },
  removeItem: function (key: string) {
    delete this.store[key]
  },
  clear: function () {
    this.store = {}
  },
}

Object.defineProperty(global, 'localStorage', {
  value: localStorageMock,
})

// Mock window.location
Object.defineProperty(global, 'location', {
  value: {
    protocol: 'https:',
    host: 'overcookied.example.com',
    href: 'https://overcookied.example.com/',
  },
  writable: true,
})
