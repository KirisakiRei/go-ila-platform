import type {
  IPrimitivePaneRenderer,
  IPrimitivePaneView,
  ISeriesPrimitive,
  SeriesAttachedParameter,
  Time,
} from 'lightweight-charts'

/**
 * ZoneSpec adalah satu area harga yang digambar di belakang candle.
 *
 * Dipakai untuk Fair Value Gap. Batas waktu opsional: bila kosong, kotak
 * membentang sampai tepi chart agar tetap terlihat walau lilin sumbernya sudah
 * keluar dari layar.
 */
export type ZoneSpec = {
  from: number
  to: number
  color: string
  borderColor?: string
  label?: string
  /** Unix detik; kosong berarti mulai dari tepi kiri. */
  timeFrom?: number
  /** Unix detik; kosong berarti sampai tepi kanan. */
  timeTo?: number
}

type DrawTarget = Parameters<IPrimitivePaneRenderer['draw']>[0]

class ZoneRenderer implements IPrimitivePaneRenderer {
  constructor(
    private readonly zones: readonly ZoneSpec[],
    private readonly series: SeriesAttachedParameter<Time>['series'],
    private readonly chart: SeriesAttachedParameter<Time>['chart'],
  ) {}

  draw(target: DrawTarget): void {
    target.useMediaCoordinateSpace(({ context, mediaSize }) => {
      const timeScale = this.chart.timeScale()

      for (const zone of this.zones) {
        const high = this.series.priceToCoordinate(Math.max(zone.from, zone.to))
        const low = this.series.priceToCoordinate(Math.min(zone.from, zone.to))
        if (high === null || low === null) continue

        const left = zone.timeFrom
          ? timeScale.timeToCoordinate(zone.timeFrom as Time)
          : 0
        const right = zone.timeTo
          ? timeScale.timeToCoordinate(zone.timeTo as Time)
          : mediaSize.width

        const x1 = Math.max(0, left ?? 0)
        const x2 = Math.min(mediaSize.width, right ?? mediaSize.width)
        const y1 = Math.min(high, low)
        const y2 = Math.max(high, low)
        if (x2 <= x1 || y2 - y1 < 0.5) continue

        context.fillStyle = zone.color
        context.fillRect(x1, y1, x2 - x1, y2 - y1)

        if (zone.borderColor) {
          context.strokeStyle = zone.borderColor
          context.lineWidth = 1
          context.setLineDash([4, 3])
          context.strokeRect(x1 + 0.5, y1 + 0.5, x2 - x1 - 1, y2 - y1 - 1)
          context.setLineDash([])
        }

        if (zone.label) {
          context.font = '10px ui-sans-serif, system-ui, sans-serif'
          context.fillStyle = zone.borderColor ?? zone.color
          context.fillText(zone.label, x1 + 6, y1 + 12)
        }
      }
    })
  }
}

class ZonePaneView implements IPrimitivePaneView {
  constructor(private readonly renderer_: IPrimitivePaneRenderer) {}

  zOrder(): 'bottom' {
    return 'bottom'
  }

  renderer(): IPrimitivePaneRenderer {
    return this.renderer_
  }
}

/**
 * ZonePrimitive menggambar kotak harga (mis. FVG) di belakang candle.
 *
 * Lightweight-charts tidak menyediakan bentuk kotak, sehingga area digambar
 * lewat primitive resmi yang punya akses ke konverter harga dan waktu yang sama
 * dengan series induknya.
 */
export class ZonePrimitive implements ISeriesPrimitive<Time> {
  private series: SeriesAttachedParameter<Time>['series'] | null = null
  private chart: SeriesAttachedParameter<Time>['chart'] | null = null
  private requestUpdate: (() => void) | null = null

  constructor(private zones: readonly ZoneSpec[]) {}

  attached(param: SeriesAttachedParameter<Time>): void {
    this.series = param.series
    this.chart = param.chart
    this.requestUpdate = param.requestUpdate
  }

  detached(): void {
    this.series = null
    this.chart = null
    this.requestUpdate = null
  }

  /** updateZones mengganti isi zona tanpa memasang ulang primitive. */
  updateZones(zones: readonly ZoneSpec[]): void {
    this.zones = zones
    this.requestUpdate?.()
  }

  paneViews(): readonly IPrimitivePaneView[] {
    if (!this.series || !this.chart) return []
    return [new ZonePaneView(new ZoneRenderer(this.zones, this.series, this.chart))]
  }
}
