# GAME.EXE port status

Every function defined in `u6-re/src/*.c`, with its Go name in `internal/game`.
"ported" means a Go function carries its `// C:` comment and no `TODO(port)` names it; "TODO" means it is missing or a `TODO(port)` still names it.
Ported: 439. TODO: 37.

## seg_0903.c

| C function | status | Go |
|---|---|---|
| LoadFileGr (C_0903_033A) | TODO |  |
| C_0903_0430 | TODO |  |
| C_0903_04DB | ported | loadTileTables |
| C_0903_070B | TODO |  |
| C_0903_0776 | TODO |  |
| C_0903_07C4 | TODO |  |
| PaletteAnimation (C_0903_0820) | ported | paletteAnimation |
| C_0903_09BB | TODO |  |
| C_0903_0A1E | ported | goToEnding |
| C_0903_0AE6 | TODO |  |
| cfg_parse (C_0903_0B46) | TODO |  |

## seg_0A33.c

| C function | status | Go |
|---|---|---|
| SubKarma (C_0A33_0008) | ported | subKarma |
| AddKarma (C_0A33_0027) | TODO |  |
| SetTileAnimation (C_0A33_0049) | ported | setTileAnimation |
| C_0A33_0073 | ported | animateTiles |
| C_0A33_0931 | ported | drawClockFace |
| C_0A33_09CE | ported | drawMap |
| OtherAnimations (C_0A33_0CBC) | ported | OtherAnimations |
| C_0A33_0D06 | ported | updateViewNow |
| TerrainDamage (C_0A33_0D1D) | ported | terrainDamage |
| C_0A33_0FEB | ported | checkStepOn |
| C_0A33_117A | ported | erupt |
| C_0A33_121A | ported | updateMoongates |
| C_0A33_12F6 | ported | changeWind |
| C_0A33_1355 | ported | passTime |
| RefreshStatus (C_0A33_1AB7) | ported | RefreshStatus |
| SetActive (C_0A33_1B36) | ported | SetActive |
| SetPartyMode (C_0A33_1BB2) | ported | SetPartyMode |
| __GetHex (C_0A33_1C4C) | ported | getHex |
| C_0A33_1CB4 | ported | Play, switchMember, command |

## seg_0C9C.c

| C function | status | Go |
|---|---|---|
| CON_mouseOffAt (C_0C9C_0007) | TODO |  |
| CON_mouseOn (C_0C9C_0050) | TODO |  |
| MkDirection (C_0C9C_0069) | ported | pointerDirection |
| C_0C9C_01FB | ported | updatePointerShape |
| C_0C9C_0286 | ported | putCharAt |
| C_0C9C_02F1 | ported | scrollUp |
| C_0C9C_0397 | ported | restoreGame |
| C_0C9C_042A | ported | loadGame |
| C_0C9C_089F | ported | saveGame |
| C_0C9C_0BCE | ported | drawTileOver |
| C_0C9C_0C10 | ported | drawSmallChar |
| C_0C9C_0C7D | ported | drawCount |
| C_0C9C_0D31 | ported | drawObjectCount |
| C_0C9C_0E57 | ported | drawTileOver |
| CON_prompt (C_0C9C_0E6E) | ported | prompt |
| C_0C9C_0F52 | ported | echo |
| __0C9C_102A | TODO |  |
| mkMouseDir (C_0C9C_1054) | ported | mouseDirection |
| mkMouseSelection (C_0C9C_1107) | ported | mouseSelection |
| do_CMD_91 (C_0C9C_121F) | ported | clickPartyView |
| do_CMD_90 (C_0C9C_12B7) | ported | clickPortraitView |
| do_CMD_92 (C_0C9C_131D) | ported | clickEquipmentView |
| do_CMD_A1 (C_0C9C_15EB) | ported | clickSpellbook |
| setDefaultCommand (C_0C9C_166D) | ported | setDefaultCommand |
| mkMouseCommand (C_0C9C_1714) | ported | mouseCommand |
| __Incant2Spell (C_0C9C_18D8) | ported | incantationSpell |
| C_0C9C_1991 | ported | selectSpell |
| C_0C9C_19D6 | ported | spellbookArrowKey |
| C_0C9C_1AE5 | ported | drawStatusCursor |
| C_0C9C_1D59 | ported | readCommand, selectTarget, moveEquipmentCursor, statusKey, spellbookKey, incantationKey |
| C_0C9C_2A16 | ported | turnedPage |
| CON_getch (C_0C9C_2A59) | ported | getch, spellbookCommand |
| CON_gets (C_0C9C_2D3A) | ported | gets |
| CON_createClip (C_0C9C_2E09) | ported | createClip |
| CON_setClip (C_0C9C_2E54) | ported | setClip |
| CON_gotoxy (C_0C9C_2E8F) | ported | gotoxy |
| CON_putch0 (C_0C9C_2EFA) | ported | putch0 |
| C_0C9C_32DE | ported | putchRepeat |
| C_0C9C_32F8 | ported | putsRaw |
| CON_atoi (C_0C9C_331A) | ported | formatNumber |
| CON_putch (C_0C9C_33D3) | ported | putch |
| CON_puts (C_0C9C_3621) | ported | puts |
| CON_printf (C_0C9C_3643) | ported | printf |

## seg_101C.c

| C function | status | Go |
|---|---|---|
| cacheChunk (C_101C_000F) | TODO |  |
| C_101C_00D7 | TODO |  |
| C_101C_0306 | ported | refreshArea |
| C_101C_052D | TODO |  |
| C_101C_054C | ported | enterArea |
| PartyEnter (C_101C_0572) | ported | partyEnter |
| PartyExit (C_101C_077E) | ported | partyExit |
| PartyTeleport (C_101C_0828) | ported | partyTeleport |
| C_101C_089E | ported | climb |
| GateTravel (C_101C_0A3A) | ported | gateTravel |
| C_101C_0ACC | ported | resurrectParty |
| getChunkIdxAt (C_101C_0CF7) | ported | chunkAt |
| GetTileAtXYZ (C_101C_0DC9) | ported | GetTileAtXYZ |

## seg_1100.c

| C function | status | Go |
|---|---|---|
| __IsVisible (C_1100_000B) | ported | isVisible |
| __1100_009F | TODO |  |
| __SetVisited (C_1100_00CF) | ported | queue |
| C_1100_0131 | ported | computeLight |
| C_1100_0306 | ported | computeView, wallSeen |

## seg_1184.c

| C function | status | Go |
|---|---|---|
| __1184_000E | TODO |  |
| OBJ_MakeDirFrame (C_1184_0073) | ported | directionFrame |
| OBJ_getHead (C_1184_0153) | ported | objectHead |
| C_1184_02FA | ported | listPosition |
| NextLoc (C_1184_053F) | ported | NextLoc |
| FindLoc (C_1184_075A) | ported | FindLoc |
| C_1184_07A7 | ported | findTypeAt |
| FindInv (C_1184_0840) | ported | FindInv |
| NextInv (C_1184_0899) | ported | NextInv |
| NextArea (C_1184_090D) | ported | NextArea |
| SearchArea (C_1184_09DE) | ported | SearchArea |
| NextInvType (C_1184_0A37) | ported | NextInvType |
| FindInvType (C_1184_0AE2) | ported | FindInvType |
| C_1184_0B3D | ported | livingAt |
| C_1184_0BC1 | ported | updateSlimeFrames |
| C_1184_0CBE | ported | moveSerpent |
| C_1184_10F1 | ported | dragonPart |
| C_1184_116D | ported | moveBigCreature |
| AddObj (C_1184_134C) | ported | AddMapObj |
| AddMapObj (C_1184_13A7) | ported | AddMapObj |
| AddInvObj (C_1184_14A8) | ported | AddInvObj |
| AddMonster (C_1184_1540) | ported | AddMonster |
| C_1184_1665 | ported | previousInList |
| C_1184_16D5 | ported | unlinkFromArea |
| DeleteObj (C_1184_17A5) | ported | DeleteObj |
| C_1184_19AA | ported | forgetFarObjects |
| C_1184_1B3A | ported | dropBody |
| C_1184_1C52 | ported | raiseBody |
| C_1184_1D2E | ported | isHeldBy |
| C_1184_1D60 | ported | lastHeld |
| MoveObj (C_1184_1DAF) | ported | MoveObj |
| InsertObj (C_1184_20DD) | ported | InsertObj |
| QuanType (C_1184_21C7) | ported | quantityType |
| C_1184_21FD | ported | amountOf |
| GiveObj (C_1184_2241) | ported | giveObj |
| TakeObj (C_1184_23F8) | ported | takeObj |
| TestObj (C_1184_252D) | ported | countObj |
| TransferObj (C_1184_257D) | TODO |  |
| C_1184_26E2 | ported | deleteTmpFiles |
| C_1184_2722 | ported | writeBlock |
| C_1184_2888 | ported | unloadObjects, unloadRegion |
| C_1184_29C4 | ported | compareListOrder |
| C_1184_2B27 | ported | moveHeldAfterHolders |
| C_1184_2DEF | ported | readBlock, loadRegion |
| C_1184_2ECC | ported | rebuildAreaObjects |
| C_1184_2FB3 | ported | unloadLevel |
| C_1184_30A6 | ported | loadLevel |
| LoadNewRegions (C_1184_3176) | ported | LoadNewRegions |
| C_1184_33CA | ported | commitObjects |
| ShowObject (C_1184_347C) | ported | showObject |
| C_1184_35EA | ported | showBigObject |
| ShowObjects (C_1184_36C2) | ported | ShowObjects |
| C_1184_3B1D | ported | resetObjectLists |
| C_1184_3B7D | ported | linkCharacters |
| GetTileString (C_1184_3C1B) | ported | tileString |
| GetObjectString (C_1184_3C9C) | ported | objectString |

## seg_155D.c

| C function | status | Go |
|---|---|---|
| C_155D_000C | ported | partyView |
| STAT_refreshComMode (C_155D_01BC) | ported | refreshCombatMode |
| C_155D_028A | ported | portraitView |
| STAT_GetEquipSlot (C_155D_0516) | ported | equipSlot |
| C_155D_063A | ported | isLight |
| GetWeight (C_155D_0661) | ported | weightOf |
| Encumbrance (C_155D_0707) | ported | encumbrance |
| C_155D_0748 | ported | weighInventory |
| C_155D_07E0 | ported | weighInventory |
| C_155D_08F4 | ported | drawEquipment |
| C_155D_0CB6 | ported | roundWeight |
| C_155D_0CF5 | ported | drawInventory |
| C_155D_1022 | ported | refreshInventory |
| C_155D_1065 | ported | equipmentView |
| C_155D_120A | ported | scrollInventory |
| C_155D_123E | ported | scrollInventory |
| C_155D_1267 | ported | pointInventory |
| C_155D_130E | ported | pointEquipment |
| C_155D_13F4 | ported | slotOf |
| __155D_141B | TODO |  |
| C_155D_144B | ported | ready |
| C_155D_1666 | ported | unreadyCheck |
| C_155D_16E7 | ported | stopWornEffect |
| C_155D_1738 | ported | unready |

## seg_16E1.c

| C function | status | Go |
|---|---|---|
| TALK_initTalk (C_16E1_0007) | ported | initTalk |
| TALK_talkTo (C_16E1_015C) | ported | talkTo |

## seg_1703.c

| C function | status | Go |
|---|---|---|
| C_1703_00A0 | ported | npcName |
| C_1703_0116 | ported | characterName |
| C_1703_0153 | ported | isOnScreen |
| C_1703_028B | ported | countNearby |
| JoinParty (C_1703_02B6) | ported | joinParty |
| C_1703_04C6 | ported | partyMemberHolding |
| C_1703_1494 | ported | indexedAddress |
| parse_statement (C_1703_1C45) | ported | parseStatement |
| C_1703_1D01 | ported | matchKeywords |
| TalkDriver (C_1703_1E23) | ported | talkDriver, converse |

## seg_1944.c

| C function | status | Go |
|---|---|---|
| C_1944_0054 | ported | canTurnInvisible |
| C_1944_0085 | ported | spellFlash |
| C_1944_01EC | ported | pushedAway, strengthResists |
| C_1944_0232 | ported | resistsByStrength |
| C_1944_0278 | ported | dodges |
| C_1944_02BE | ported | reagentCount |
| SPELL_RefreshList (C_1944_0321) | ported | refreshSpellList |
| SPELL_DrawBG (C_1944_05B1) | ported | drawSpellbook |
| C_1944_06CA | ported | pageRow |
| TakeReagents (C_1944_06EF) | ported | takeReagents |
| SPELL_RefreshPage (C_1944_0773) | ported | refreshPage |
| C_1944_0A43 | ported | heldOfQuality |
| C_1944_0AA9 | ported | isDoor |
| C_1944_0AD0 | ported | sleepSpell |
| C_1944_0B32 | ported | poison |
| C_1944_0B85 | ported | magicArrow |
| C_1944_0BF4 | ported | drainSpell |
| C_1944_0CA3 | ported | charm |
| C_1944_0D46 | ported | killSpell |
| C_1944_0DCF | ported | douseOrIgnite |
| C_1944_0F02 | ported | nearestFoe |
| C_1944_1038 | ported | chainBolt |
| C_1944_114D | ported | healSpell |
| C_1944_11BB | ported | dispel |
| C_1944_1232 | ported | burn |
| C_1944_12FC | ported | aimSpell, replicate, clone, dispelField |
| C_1944_1A42 | ported | resurrect |
| C_1944_1C51 | ported | resurrectSpell |
| __1944_1DAB | TODO |  |
| C_1944_1DF4 | ported | touchSpell |
| C_1944_22F4 | ported | slime |
| C_1944_23F2 | ported | reappear |
| C_1944_256A | ported | areaSpell |
| C_1944_27AF | ported | fieldSpell |
| C_1944_29AA | ported | frighten |
| C_1944_2A15 | ported | surroundingSpell |
| C_1944_2B9A | ported | createFood |
| C_1944_2BFD | ported | summon |
| C_1944_2DA7 | ported | telekinesis |
| C_1944_305A | ported | gateTravelSpell |
| C_1944_310C | ported | xrayVision |
| C_1944_3175 | ported | pickpocket |
| C_1944_34CF | ported | insectSwarm |
| C_1944_35EC | ported | webSpell |
| C_1944_3750 | ported | enchant |
| C_1944_39A1 | ported | wizardEye |
| C_1944_3B2D | ported | tremor |
| C_1944_3C30 | ported | seanceSpell |
| C_1944_3D0F | ported | hailStormSpell |
| C_1944_3DB8 | ported | windSpell |
| C_1944_3F6C | ported | wingStrike |
| C_1944_4260 | ported | selfInvisibility |
| C_1944_42AC | ported | locate |
| C_1944_435B | ported | windChange |
| C_1944_43A3 | ported | armageddonSpell |
| C_1944_43D5 | ported | castSpell |
| C_1944_4926 | ported | castFrom |
| C_1944_4C2F | ported | cast |

## seg_1E0F.c

| C function | status | Go |
|---|---|---|
| C_1E0F_000F | ported | passable |
| IsOutOfArea (C_1E0F_056E) | ported | isOutOfArea |
| C_1E0F_05D8 | ported | blockedAt |
| C_1E0F_0664 | ported | turnTo |
| SubMov (C_1E0F_0F54) | ported | subMove |
| C_1E0F_0FA9 | ported | draggedUnderTurn |
| C_1E0F_1056 | ported | alreadyNear |
| MoveFollowers (C_1E0F_1193) | ported | MoveFollowers |
| Board (C_1E0F_14BE) | ported | board |
| Unboard (C_1E0F_1745) | ported | unboard |
| C_1E0F_184D | ported | enterGates, travelThrough |
| C_1E0F_1B0E | ported | advance |
| C_1E0F_2001 | ported | openDoorUnder |
| C_1E0F_2125 | ported | plateNear |
| C_1E0F_2184 | ported | furnitureAt |
| __AtDestination (C_1E0F_2276) | ported | atDestination |
| C_1E0F_25F9 | ported | tracePath |
| C_1E0F_291C | ported | sendHome |
| C_1E0F_2A74 | ported | visit |
| __ResetPath (C_1E0F_2C9D) | ported | reversePath |
| C_1E0F_2D37 | ported | findPath |
| C_1E0F_31C7 | ported | randomBias |
| SubTerrainMov (C_1E0F_31F7) | ported | subTerrainMove |
| TryStraightMove (C_1E0F_32D6) | ported | TryStraightMove |
| C_1E0F_33C4 | ported | loiter |
| __TryDiagMove (C_1E0F_3470) | ported | tryDiagonalMove |
| TryMoveTo (C_1E0F_35A7) | ported | TryMoveTo |
| C_1E0F_37DB | ported | wander |
| __DoOnPath (C_1E0F_387D) | ported | followPath |
| C_1E0F_39B3 | ported | approachParty |
| C_1E0F_3B60 | ported | arrest |
| C_1E0F_3D8D | ported | stealGold |
| C_1E0F_3E08 | ported | talkToParty |
| __NewPathIndex (C_1E0F_3E27) | ported | newPathSlot |
| C_1E0F_3E6A | ported | npcTurn |
| C_1E0F_4265 | ported | addResistance |
| __ComputeResistance (C_1E0F_436A) | ported | computeResistance |
| C_1E0F_464A | ported | schedulePaths |
| C_1E0F_4746 | ported | computeGravity |
| C_1E0F_4AFA | ported | avatarDied |
| C_1E0F_4B6A | ported | checkActiveMember |
| C_1E0F_4E0A | ported | runTurns |
| C_1E0F_512A | ported | startTurns |
| C_1E0F_5165 | ported | followSchedules |

## seg_2337.c

| C function | status | Go |
|---|---|---|
| COMBAT_getCathesus (C_2337_000B) | ported | distance |
| COMBAT_stopCharm (C_2337_005E) | ported | stopCharm |
| COMBAT_begin (C_2337_00C8) | ported | beginCombat |
| COMBAT_breakOff (C_2337_01BC) | ported | breakOff |
| COMBAT_TryTeleport (C_2337_02DC) | ported | tryTeleport |
| GetStr (C_2337_03E7) | ported | strengthOf |
| GetDex (C_2337_0411) | ported | dexterityOf |
| GetInt (C_2337_0458) | ported | intelligenceOf |
| COMBAT_TestIntel (C_2337_0482) | ported | testIntelligence |
| ComputeLevel (__2337_04BE) | TODO |  |
| GetLevel (C_2337_04E1) | ported | portraitView |
| AddXP (C_2337_04F4) | ported | addExperience |
| MaxHP (C_2337_0529) | ported | maxHitPoints |
| MaxMagic (C_2337_0559) | ported | maxMagic |
| SetTypeUnconscious (C_2337_05B4) | ported | setUnconscious |
| COMBAT_canSee (C_2337_0837) | ported | canSee |
| C_2337_08F1 | ported | objectAtForSelection |
| GetArmorPoints (C_2337_0983) | ported | armorPoints |
| __GetProtection (C_2337_09BF) | ported | protection |
| __GetWeaponRange (C_2337_0A14) | ported | weaponRange |
| C_2337_0A4E | ported | interferes |
| COMBAT_Missile (C_2337_0B3F) | ported | throw |
| C_2337_0B99 | ported | rangedAttack |
| GetDamagePoints (C_2337_10E4) | ported | damagePoints |
| CloneObject (C_2337_115F) | ported | cloneObject |
| COMBAT_showHit (C_2337_121F) | ported | showHit |
| LooseHP (C_2337_12F5) | ported | loseHitPoints, kill, divideSlime, showHitPoints |
| __AttackTest (C_2337_18CF) | ported | hits |
| COMBAT_displaySeverity (C_2337_198F) | ported | displaySeverity |
| C_2337_1A99 | ported | pickItem |
| C_2337_1B0E | ported | reportCrime |
| C_2337_1C1E | ported | provoke |
| __DoHit (C_2337_1D20) | ported | doHit |
| COMBAT_getHead (C_2337_205C) | ported | combatHead |
| __TryHit (C_2337_214D) | ported | tryHit |
| C_2337_27C5 | ported | listWeapons |
| C_2337_2875 | ported | attackWithNext |
| COMBAT_attack (C_2337_2AA9) | ported | attack |
| COMBAT_range (C_2337_2BEE) | ported | combatRange |
| COMBAT_pickOpponent (C_2337_2D6E) | ported | pickOpponent |
| COMBAT_AI_Assault (C_2337_2F3E) | ported | assault |
| C_2337_310E | ported | holdLine |
| COMBAT_AI_Front (C_2337_3464) | ported | front |
| COMBAT_AI_Rear (C_2337_34C9) | ported | rear |
| COMBAT_AI_Flank (C_2337_37F2) | ported | flank |
| COMBAT_AI_Bersek (C_2337_3C3A) | ported | berserk |
| COMBAT_AI_Retreat (C_2337_3ED2) | ported | retreat |
| COMBAT_AI_Like (C_2337_407F) | ported | like |
| COMBAT_AI_Unfriendly (C_2337_41B7) | ported | unfriendly |
| COMBAT_AI_Immobile (C_2337_42EE) | ported | immobile |
| __TryGrowVine (C_2337_43D2) | ported | growTangle |
| COMBAT_AI_Tangle (C_2337_44F6) | ported | tangle |

## seg_27a1.c

| C function | status | Go |
|---|---|---|
| C_27A1_0008 | ported | memberUser |
| C_27A1_0028 | ported | userX |
| C_27A1_0051 | ported | trap |
| C_27A1_0082 | ported | isContainer |
| C_27A1_00A9 | ported | canPutInto |
| C_27A1_012C | ported | drop |
| IsOkToGet (C_27A1_0153) | ported | okToGet |
| SetOkToGet (C_27A1_019E) | ported | setOkToGet |
| C_27A1_01DE | ported | use |
| C_27A1_0205 | ported | parseNumber |
| C_27A1_02D9 | ported | showCharacter |
| C_27A1_061E | ported | printArticle |
| C_27A1_066D | ported | canRead |
| C_27A1_06A2 | ported | canRead |
| C_27A1_06D7 | ported | canRead |
| C_27A1_078F | ported | readBook |
| C_27A1_0841 | ported | printCount |
| C_27A1_08D3 | ported | isClosed |
| C_27A1_0919 | ported | objectBelow |
| C_27A1_09A1 | ported | search |
| C_27A1_0C67 | ported | look, lookSpellbook |
| C_27A1_1330 | ported | supports |
| C_27A1_13BE | ported | howMany |
| C_27A1_14DA | ported | drop |
| C_27A1_18F5 | ported | get |
| C_27A1_1DAB | ported | cannotPush |
| C_27A1_1E69 | ported | carrierOf |
| C_27A1_1E8B | ported | move |
| C_27A1_28A3 | ported | trap |
| C_27A1_2A44 | ported | useDoor |
| C_27A1_2BBC | ported | useChest |
| C_27A1_2D34 | ported | skillFails |
| C_27A1_2D61 | ported | gazeCrystalBall |
| C_27A1_2D8E | ported | useKey |
| C_27A1_2FD1 | ported | lightTorch |
| C_27A1_319F | ported | useGem |
| C_27A1_31F6 | ported | light |
| C_27A1_32FA | ported | toggleLit |
| C_27A1_335A | ported | playInstrument |
| C_27A1_338D | ported | ringBell |
| C_27A1_3425 | ported | buryMoonstone |
| C_27A1_3537 | ported | useStaff |
| __27A1_3617 | TODO |  |
| __27A1_363C | TODO |  |
| C_27A1_3661 | ported | fillBucket |
| C_27A1_36E7 | ported | useObject |
| C_27A1_376D | ported | churn |
| C_27A1_37D3 | ported | fill |
| C_27A1_3832 | ported | drinkPotion |
| C_27A1_3A35 | ported | fireCannon |
| C_27A1_3B59 | ported | lightKeg |
| AfterExplosion (C_27A1_3BE1) | ported | afterExplosion |
| Detonate (C_27A1_3D2B) | ported | detonate |
| __SearchTypeAt (C_27A1_3E21) | ported | typeAt |
| C_27A1_3E59 | ported | thingOnBridge |
| C_27A1_3E9A | ported | characterAt |
| C_27A1_3ED1 | ported | isBank |
| C_27A1_3EF6 | ported | addFixed |
| C_27A1_3F47 | ported | moveBridge |
| C_27A1_433D | ported | turnCrank |
| C_27A1_4479 | ported | switchLever |
| C_27A1_4672 | ported | useObject |
| C_27A1_47E0 | ported | useObject |
| C_27A1_47F3 | ported | buildBalloon |
| C_27A1_49C3 | ported | inflateBalloon |
| C_27A1_4B0B | ported | destroyGargoyleEggs |
| C_27A1_4B98 | ported | useRune |
| C_27A1_4D46 | ported | fish |
| C_27A1_4E6F | ported | drink |
| C_27A1_4E9B | ported | wish |
| C_27A1_4FC6 | ported | dig |
| C_27A1_51C4 | ported | landNear |
| C_27A1_5289 | ported | useVehicle |
| C_27A1_5431 | ported | mount |
| C_27A1_5503 | ported | mountHorse |
| C_27A1_55F0 | ported | dismount |
| C_27A1_5789 | ported | useOrb |
| C_27A1_5935 | ported | gazeAt, gazeCrystalBall |
| C_27A1_5A97 | ported | lookThrough |
| C_27A1_5BCF | ported | blowHorn |
| C_27A1_5DF2 | ported | usePassthrough |
| C_27A1_5F43 | ported | eat |
| C_27A1_5FAC | ported | openVortex |
| C_27A1_60BD | ported | useObject |
| C_27A1_60D6 | ported | useObject |
| C_27A1_60F5 | ported | vehicleBlocked |
| C_27A1_6179 | ported | use |

## seg_2E2D.c

| C function | status | Go |
|---|---|---|
| __2E2D_0003 | TODO |  |
| GetMonsterClass (C_2E2D_004B) | ported | monsterClass |
| mkRandom (C_2E2D_0088) | ported | randomAround |
| C_2E2D_00BE | ported | equipCreature |
| EGG_generate (C_2E2D_0499) | ported | generate |
| EGG_hatches (C_2E2D_0760) | ported | hatch |
| EGG_hatchArea (C_2E2D_0DFE) | ported | hatchEggs |

## seg_2F1A.c

| C function | status | Go |
|---|---|---|
| MUS_0008 | TODO |  |
| MUS_0065 | ported | sfx |
| MUS_043B | ported | addSoundSource |
| MUS_051E | TODO |  |
| MUS_0525 | ported | playSoundSources |
| MUS_07F6 | TODO |  |
| MUS_0851 | TODO |  |
| MUS_091A | ported | playSong |
| MUS_09A8 | ported | playAreaSong |
| MUS_0A3A | TODO |  |

## seg_2FC1.c

| C function | status | Go |
|---|---|---|
| C_2FC1_000A | TODO |  |
| MagicalHalo (C_2FC1_00B0) | ported | magicalHalo |
| C_2FC1_017F | ported | grow |
| C_2FC1_0212 | ported | redrawFrame |
| C_2FC1_032B | ported | redrawForeground |
| C_2FC1_131E | ported | webMissiles |
| C_2FC1_13B0 | ported | hailStorm |
| ShakeScreen (C_2FC1_18BD) | ported | shakeScreen |
| C_2FC1_193C | ported | redrawWorld |
| C_2FC1_19A1 | ported | redrawWorld |
| C_2FC1_19C5 | ported | drawSky |
| C_2FC1_1BCE | ported | vortex |
| C_2FC1_1C19 | ported | drawPortrait |
| LoadConversation (C_2FC1_1D40) | ported | loadConversation |
| C_2FC1_1EAF | ported | drawDoll |
| peer_tile (C_2FC1_1F14) | ported | peerTile |
| C_2FC1_1FD6 | ported | peer |

## seg_3200.c

| C function | status | Go |
|---|---|---|
| IsEdible (C_3200_000E) | ported | eatFrom |
| TryEatExclusive (C_3200_0035) | ported | eatFrom |
| TryEat (C_3200_0094) | ported | feed |
| C_3200_00D3 | ported | notWilderness |
| C_3200_0160 | ported | foesNear |
| C_3200_0226 | ported | othersNear |
| C_3200_02C4 | ported | noRoom |
| HasNightEncounter (C_3200_0309) | ported | nightAmbush |
| C_3200_0467 | ported | heal |
| C_3200_0514 | ported | repairHull |
| C_3200_055D | ported | rest |

## seg_32C3.c

| C function | status | Go |
|---|---|---|
| DISK_0005 | TODO |  |
| DISK_0008 | TODO |  |
| DISK_0053 | TODO |  |
| DISK_confirm (DISK_0097) | TODO |  |
| DISK_0202 | TODO |  |
| DISK_full (DISK_0211) | TODO |  |
| DISK_init (DISK_0266) | TODO |  |
