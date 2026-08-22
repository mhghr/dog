import type { Icon, IconProps } from "@phosphor-icons/react";
import {
  Broadcast as BroadcastBase,
  Browser as BrowserBase,
  CalendarCheck as CalendarCheckBase,
  CaretLeft as CaretLeftBase,
  Clock as ClockBase,
  EnvelopeSimple as EnvelopeSimpleBase,
  Gauge as GaugeBase,
  GearSix as GearSixBase,
  Globe as GlobeBase,
  List as ListBase,
  MagnifyingGlass as MagnifyingGlassBase,
  MapPin as MapPinBase,
  Monitor as MonitorBase,
  Moon as MoonBase,
  PlugsConnected as PlugsConnectedBase,
  Pulse as PulseBase,
  ShieldCheck as ShieldCheckBase,
  ShieldWarning as ShieldWarningBase,
  SignOut as SignOutBase,
  SquaresFour as SquaresFourBase,
  Sun as SunBase,
  Timer as TimerBase,
  CellTower as CellTowerBase,
  Tray as TrayBase,
  TreeStructure as TreeStructureBase,
  UserCircle as UserCircleBase,
  Warning as WarningBase,
} from "@phosphor-icons/react/dist/ssr";

// Design-system icon source of truth. The SSR entrypoint is context-free, so
// these components are safe in both server and client components; the regular
// weight is baked in here instead of a client-only IconContext.
function regular(Base: Icon): Icon {
  function RegularIcon(props: IconProps) {
    return <Base weight="regular" {...props} />;
  }
  return RegularIcon as Icon;
}

export const Broadcast = regular(BroadcastBase);
export const Browser = regular(BrowserBase);
export const CalendarCheck = regular(CalendarCheckBase);
export const CaretLeft = regular(CaretLeftBase);
export const Clock = regular(ClockBase);
export const EnvelopeSimple = regular(EnvelopeSimpleBase);
export const Gauge = regular(GaugeBase);
export const GearSix = regular(GearSixBase);
export const Globe = regular(GlobeBase);
export const List = regular(ListBase);
export const MagnifyingGlass = regular(MagnifyingGlassBase);
export const MapPin = regular(MapPinBase);
export const Monitor = regular(MonitorBase);
export const Moon = regular(MoonBase);
export const PlugsConnected = regular(PlugsConnectedBase);
export const Pulse = regular(PulseBase);
export const ShieldCheck = regular(ShieldCheckBase);
export const ShieldWarning = regular(ShieldWarningBase);
export const SignOut = regular(SignOutBase);
export const SquaresFour = regular(SquaresFourBase);
export const Sun = regular(SunBase);
export const Timer = regular(TimerBase);
export const CellTower = regular(CellTowerBase);
export const Tray = regular(TrayBase);
export const TreeStructure = regular(TreeStructureBase);
export const UserCircle = regular(UserCircleBase);
export const Warning = regular(WarningBase);

export type { Icon as AppIcon } from "@phosphor-icons/react";
