export const bouquets = [
  {
    id: 1,
    price: '12 500 ₸',
    name: 'Пудровые пионы, 15 шт',
    seller: 'Айгерим',
    rating: '4.9',
    photo: '/assets/b1.jpg',
    avatar: '/assets/a1.jpg',
    tags: ['peonies'],
  },
  {
    id: 2,
    price: '8 900 ₸',
    name: 'Полевой микс с лавандой',
    seller: 'Дильназ',
    rating: '4.8',
    photo: '/assets/b2.jpg',
    avatar: '/assets/a2.jpg',
    tags: ['wild'],
  },
  {
    id: 3,
    price: '24 000 ₸',
    name: 'Монобукет белых роз, premium',
    seller: 'Florist Studio',
    rating: '5.0',
    photo: '/assets/b3.jpg',
    avatar: '/assets/a3.jpg',
    tags: ['roses'],
  },
  {
    id: 4,
    price: '15 700 ₸',
    name: 'Композиция в крафте «Осень»',
    seller: 'Малика',
    rating: '4.7',
    photo: '/assets/b4.jpg',
    avatar: '/assets/a4.jpg',
    tags: ['composition'],
  },
  {
    id: 5,
    price: '6 200 ₸',
    name: 'Маленький букет ромашек',
    seller: 'Сабина',
    rating: '4.8',
    photo: '/assets/b5.jpg',
    avatar: '/assets/a5.jpg',
    tags: ['wild'],
  },
  {
    id: 6,
    price: '18 400 ₸',
    name: 'Пионовидные розы и эвкалипт',
    seller: 'Bloom Almaty',
    rating: '4.9',
    photo: '/assets/b6.jpg',
    avatar: '/assets/a6.jpg',
    tags: ['roses', 'peonies'],
  },
]

export const categories = [
  { key: 'all', label: 'Все' },
  { key: 'roses', label: 'Розы' },
  { key: 'peonies', label: 'Пионы' },
  { key: 'wild', label: 'Полевые' },
  { key: 'composition', label: 'Композиции' },
  { key: 'dried', label: 'Сухоцветы' },
]

export function filterByCategory(cat) {
  if (!cat || cat === 'all') return bouquets
  return bouquets.filter((b) => b.tags.includes(cat))
}
