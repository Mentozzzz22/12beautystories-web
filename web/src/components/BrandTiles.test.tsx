import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import type { Brand } from '../api/types'
import { BrandTiles } from './BrandTiles'

const brand = (slug: string, name: string, logoUrl = ''): Brand => ({
  id: slug.length,
  slug,
  name,
  description: '',
  logoUrl,
  lines: [],
})

describe('BrandTiles', () => {
  it('shows the logo when there is one and the name otherwise', () => {
    render(
      <MemoryRouter>
        <BrandTiles
          brands={[
            brand('tokio-inkarami', 'TOKIO INKARAMI', '/uploads/tokio.jpg'),
            brand('payot', 'Payot'),
          ]}
        />
      </MemoryRouter>,
    )

    expect(screen.getByRole('img', { name: 'TOKIO INKARAMI' })).toHaveAttribute(
      'src',
      '/uploads/tokio.jpg',
    )
    expect(screen.getByRole('link', { name: 'TOKIO INKARAMI' })).toHaveAttribute(
      'href',
      '/brands/tokio-inkarami',
    )
    expect(screen.getByRole('link', { name: 'Payot' })).toHaveTextContent('Payot')
  })
})
