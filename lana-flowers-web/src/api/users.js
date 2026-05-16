import { api } from './client'

export function getMe() {
  return api('/users/me')
}

export function updateMe(patch) {
  return api('/users/me', { method: 'PATCH', body: patch })
}

export function getUser(id) {
  return api('/users/' + id)
}
